package localstore

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"gitlab.com/adrian_blx/takoprint/lib/store"
)

type LocalStoreFile struct {
	sync.RWMutex
	r    io.ReadCloser
	name string
	pos  int64
	size int64
}

// ReadFile returns a gcode stream from a configured local store.
func (ls *LocalStore) ReadFile(path, file string) (store.Stream, error) {
	path, file, err := cleanPaths(path, file)
	if err != nil {
		return nil, fmt.Errorf("invalid path")
	}

	fh, err := os.Open(filepath.Join(ls.base, path, file))
	if err != nil {
		return nil, err
	}
	return FromFilehandle(fh)
}

// FromFilehandle wraps an open FH into a gcode stream.
func FromFilehandle(fh *os.File) (store.Stream, error) {
	stat, err := fh.Stat()
	if err != nil {
		fh.Close()
		return nil, err
	}

	return &LocalStoreFile{
		r:    fh,
		name: stat.Name(),
		size: stat.Size(),
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
