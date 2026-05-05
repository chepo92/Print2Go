package bufstore

import (
	"bytes"
	"io"
	"sync"

	store "github.com/chepo92/Print2Go/storage"
)

type Bufstore struct {
	sync.RWMutex
	fh    io.Reader
	name  string
	pos   int64
	size  int64
	lines int64
}

func New(b []byte, name string) store.Stream {
	return &Bufstore{
		fh:    bytes.NewReader(b),
		name:  name,
		size:  int64(len(b)),
		lines: int64(bytes.Count(b, []byte{'\n'})),
	}
}

func (bs *Bufstore) Read(b []byte) (int, error) {
	nr, err := bs.fh.Read(b)
	bs.Lock()
	bs.pos += int64(nr)
	bs.Unlock()
	return nr, err
}

func (bs *Bufstore) Close() error {
	return nil
}

func (bs *Bufstore) Name() string {
	return bs.name
}

func (bs *Bufstore) Size() int64 {
	return bs.size
}

func (bs *Bufstore) Pos() int64 {
	bs.RLock()
	defer bs.RUnlock()
	return bs.pos
}

func (bs *Bufstore) LineCount() int64 {
	return bs.lines
}
