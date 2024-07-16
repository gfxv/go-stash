package core

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

type TransformPathFunc func(string) (string, string)

func DefaultTransformPathFunc(key string) (prefix string, name string) {
    fullHash := sha1.Sum([]byte(key))
    strHash := hex.EncodeToString(fullHash[:])
    prefix, name = strHash[:2], strHash[2:]
    return
}

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
    prefix, filename := s.transformPath(key)

    prefixPath := fmt.Sprintf("%s/%s", s.baseDir, prefix)
    if err := os.MkdirAll(prefixPath, os.ModePerm); err != nil { // TODO: change permissions (?), now - 777
        return err
    }

    fullPath := fmt.Sprintf("%s/%s/%s", s.baseDir, prefix, filename)
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
