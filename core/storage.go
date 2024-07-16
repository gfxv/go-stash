package core

import (
	"fmt"
	"io"
	"os"
)

type TransformPathFunc func(string) (string, error)

type Storage struct {
	baseDir       string
	transformPath TransformPathFunc
}

func NewDefaultStorage() *Storage {
	return &Storage{}
}

func (s *Storage) WithBaseDir(baseDir string) *Storage {
	s.baseDir = baseDir
	return s
}

func (s *Storage) WithTransformPathFunc(pathFunc TransformPathFunc) *Storage {
	s.transformPath = pathFunc
	return s
}

func (s *Storage) Read(key string) {

}

func (s *Storage) Write(key string, r io.Reader) error {
    pathKey, err := s.transformPath(key)
    if err != nil {
        return err
    }

    fullPath := fmt.Sprintf("%s/%s", s.baseDir, pathKey)
    if err = os.MkdirAll(fullPath, os.ModePerm); err != nil { // TODO: change permissions (?), now - 777
        return err
    }

    f, err := os.Create(fullPath)
    if err != nil {
        return err
    }
    
    // Copy() returns (written, err), written is ignored
    _, err = io.Copy(f, r)
    if err != nil {
        return err
    }

    return nil
}

func (s *Storage) Delete() {}

func (s *Storage) DeleteAll() {}
