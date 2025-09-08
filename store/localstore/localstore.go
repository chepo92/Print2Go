package localstore

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type LocalStore struct {
	base string
}

func New(base string) *LocalStore {
	return &LocalStore{
		base: base,
	}
}

// UploadFile uploads a gcode file to the specified path in a local store.
func (ls *LocalStore) UploadFile(path, file string, r io.ReadCloser) error {
	path, file, err := cleanPaths(path, file)
	if err != nil {
		return fmt.Errorf("Invalid path")
	}

	os.MkdirAll(filepath.Join(ls.base, path), 0755)
	fh, err := os.Create(filepath.Join(ls.base, path, file))
	if err != nil {
		return fmt.Errorf("Failed to create file: %v", err)
	}
	defer fh.Close()
	io.Copy(fh, r)
	return nil
}

func cleanPaths(path, file string) (string, string, error) {
	if path != "" {
		path = filepath.Clean(path)
	}
	file = filepath.Clean(file)

	if file == "" {
		return "", "", fmt.Errorf("Invalid path")
	}
	return path, file, nil
}
