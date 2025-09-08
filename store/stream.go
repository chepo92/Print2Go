package store

import (
	"io"
)

// Stream is an interface for a gcode stream
type Stream interface {
	// underlying reader
	io.ReadCloser
	// total size of the stream
	Size() int64
	// current read position
	Pos() int64
	// name of the stream, usually the filename?
	Name() string
}
