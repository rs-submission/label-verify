package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type BlobStore interface {
	Put(id string, data []byte) (ref string, err error)
	Get(ref string) ([]byte, error)
	Delete(ref string) error
}

type FileStore struct {
	root string
}

func NewFileStore(root string) *FileStore {
	if root == "" {
		root = "./data/images"
	}
	return &FileStore{root: root}
}

func (s *FileStore) Put(id string, data []byte) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", fmt.Errorf("id is required")
	}
	name := filepath.Base(id)
	if name == "." || name == string(filepath.Separator) {
		return "", fmt.Errorf("invalid id %q", id)
	}
	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(s.root, name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", err
	}
	return name, nil
}

func (s *FileStore) Get(ref string) ([]byte, error) {
	name := filepath.Base(ref)
	if name != ref || strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("invalid ref %q", ref)
	}
	return os.ReadFile(filepath.Join(s.root, name))
}

func (s *FileStore) Delete(ref string) error {
	name := filepath.Base(ref)
	if name != ref || strings.TrimSpace(name) == "" {
		return fmt.Errorf("invalid ref %q", ref)
	}
	err := os.Remove(filepath.Join(s.root, name))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
