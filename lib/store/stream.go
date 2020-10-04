package store

import (
	"io"
)

type Stream interface {
	io.ReadCloser
	Size() int64
	Pos() int64
	Name() string
}
