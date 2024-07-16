package core

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
)

type TransformPathFunc func(string) (string, string)

func DefaultTransformPathFunc(key string) (prefix string, parent string) {
    fullHash := sha1.Sum([]byte(key))
    strHash := hex.EncodeToString(fullHash[:])
    prefix, parent = strHash[:5], strHash[5:]
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

func (s *Storage) Has(path string) bool {
    _, err := os.Stat(path)
    return !errors.Is(err, os.ErrNotExist)
}

func (s *Storage) Read(key string) {

}

func (s *Storage) Write(key string, data []byte) error {
    prefix, parent := s.transformPath(key)

    folders := fmt.Sprintf("%s/%s/%s", s.baseDir, prefix, parent)
    if err := os.MkdirAll(folders, os.ModePerm); err != nil { // TODO: change permissions (?), now - 777
        return err
    }

    contentPath := buildContentPath(data) 
    fullPath := fmt.Sprintf("%s/%s/%s/%s", s.baseDir, prefix, parent, contentPath)

    if s.Has(fullPath) {
        return fmt.Errorf("collision detected! \nkey '%s' with data provided already exists", key)
    }

    f, err := os.Create(fullPath)
    if err != nil {
        return err
    }

    // Copy() returns (written, err), written is ignored
    _, err = io.Copy(f, bytes.NewReader(data))
    if err != nil {
        return err
    }

    return nil
}

func buildContentPath(data []byte) string {
    fileHash := sha1.Sum(data)
    return hex.EncodeToString(fileHash[:])
}

func (s *Storage) Delete() {}

func (s *Storage) DeleteAll() {}
