package localstore

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/chepo92/PrintAndGo/gcode"
	"github.com/chepo92/PrintAndGo/store"
)

// LocalStoreFile implements a file stream from a local file.
type LocalStoreFile struct {
	sync.RWMutex
	r         io.ReadCloser
	name      string
	pos       int64
	size      int64
	lineCount int64
}

// ReadFile returns a gcode stream from a configured local store.
func (ls *LocalStore) ReadFile(path, file string) (store.Stream, error) {
	path, file, err := cleanPaths(path, file)
	if err != nil {
		return nil, fmt.Errorf("Invalid path")
	}

	fh, err := os.Open(filepath.Join(ls.base, path, file))
	if err != nil {
		return nil, err
	}
	return FromFilehandle(fh)
}

// FromFilehandle wraps an open filehanle into a gcode stream.
func FromFilehandle(fh *os.File) (store.Stream, error) {
	stat, err := fh.Stat()
	if err != nil {
		fh.Close()
		return nil, err
	}
	// Count lines
	numLines, err := validLineCounter(fh, gcode.GcodeFilter())
	if err != nil {
		fh.Close()
		return nil, err
	}
	fmt.Printf("File has %d Gcode valid lines\n", numLines)
	// Reset file pointer to start
	_, err = fh.Seek(0, io.SeekStart)
	if err != nil {
		fh.Close()
		return nil, err
	}

	return &LocalStoreFile{
		r:         fh,
		name:      stat.Name(),
		size:      stat.Size(),
		lineCount: int64(numLines),
	}, nil
}

func (lsf *LocalStoreFile) Read(b []byte) (int, error) {
	nr, err := lsf.r.Read(b)

	lsf.Lock()
	defer lsf.Unlock()
	lsf.pos += int64(nr)
	return nr, err
}

func (lsf *LocalStoreFile) Pos() int64 {
	lsf.RLock()
	defer lsf.RUnlock()
	return lsf.pos
}

func (lsf *LocalStoreFile) Close() error {
	return lsf.r.Close()
}

func (lsf *LocalStoreFile) Name() string {
	return lsf.name
}

func (lsf *LocalStoreFile) Size() int64 {
	return lsf.size
}

func (lsf *LocalStoreFile) LineCount() int64 {
	return lsf.lineCount
}

// optimized line counter
func optimizedLineCounter(r io.Reader) (int, error) {
	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := r.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			return count, nil

		case err != nil:
			return count, err
		}
	}
}

// validLineCounter cuenta líneas válidas usando un filtro.
// El filtro debe devolver true si la línea debe ser ignorada.
func validLineCounter(r io.Reader, filter func(string) bool) (int, error) {
	scanner := bufio.NewScanner(r)

	// Configurar buffer grande por si hay líneas largas
	const maxCapacity = 1024 * 1024 // 1 MB por línea
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, maxCapacity)

	count := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !filter(line) {
			count++
		}
	}

	if err := scanner.Err(); err != nil {
		return count, err
	}
	return count, nil
}
