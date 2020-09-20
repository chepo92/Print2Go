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

func (ls *LocalStore) UploadFile(path, file string, r io.ReadCloser) error {
	path, file, err := cleanPaths(path, file)
	if err != nil {
		return fmt.Errorf("invalid path")
	}

	os.MkdirAll(filepath.Join(ls.base, path), 0755)
	fh, err := os.Create(filepath.Join(ls.base, path, file))
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer fh.Close()
	io.Copy(fh, r)
	return nil
}

func (ls *LocalStore) ReadFile(path, file string) (*os.File, error) {
	path, file, err := cleanPaths(path, file)
	if err != nil {
		return nil, fmt.Errorf("invalid path")
	}
	return os.Open(filepath.Join(ls.base, path, file))
}

func cleanPaths(path, file string) (string, string, error) {
	if path != "" {
		path = filepath.Clean(path)
	}
	file = filepath.Clean(file)

	if file == "" {
		return "", "", fmt.Errorf("invalid path")
	}
	return path, file, nil
}
