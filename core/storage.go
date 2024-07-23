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

type TransformPathFunc func([]byte) (string, string)

func DefaultTransformPathFunc(data []byte) (prefix string, filename string) {
    fullHash := sha1.Sum(data)
    strHash := hex.EncodeToString(fullHash[:])
    prefix, filename = strHash[:5], strHash[5:]
    return
}

type Storage struct {
    baseDir       string
    transformPath TransformPathFunc
    pack          PackFunc
    unpack        UnpackFunc
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

func (s *Storage) WithCompressionFuncs(compress PackFunc, decompress UnpackFunc) *Storage {
    s.pack = compress
    s.unpack = decompress
    return s
}

func (s *Storage) Has(path string) bool {
    _, err := os.Stat(path)
    return !errors.Is(err, os.ErrNotExist)
}


// Store receives path (or multiple paths) to file or directory that should be saved on the disk
func (s *Storage) Store(key string, paths ...string) error {
    var err error
    for _, p := range paths {
        tree, err := NewFileTree(p)
        if err != nil {
            return err
        }
        err = s.saveTree(key, *tree)
    }
    return err
}

func (s *Storage) saveTree(key string, tree ...File) error {
    var err error
    for _, t := range tree {
        if t.IsDir {
            s.saveTree(key, t.Children...)
        }
        err = s.writeFile(key, t.FullPath)
    }
    return err
}

func (s *Storage) writeFile(key, path string) error {
    file, err := os.ReadFile(path)
    if err != nil {
        return err
    }
    return s.Write(key, file)
}

func (s *Storage) Read(key string) {
}

// TODO: save key to db 
// Write 
func (s *Storage) Write(key string, data []byte) error {
    prefix, filename := s.transformPath(data)

    folders := fmt.Sprintf("%s/%s", s.baseDir, prefix)
    if err := os.MkdirAll(folders, os.ModePerm); err != nil { // TODO: change permissions (?), now - 777
        return err
    }

    fullPath := fmt.Sprintf("%s/%s", folders, filename)

    if s.Has(fullPath) {
        return fmt.Errorf("collision detected! \nkey '%s' with data provided already exists", key)
    }

    file, err := os.Create(fullPath)
    if err != nil {
        return err
    }

    // write compressed data to storage
    compressed := s.pack(data)
    _, err = io.Copy(file, bytes.NewReader(compressed))
    if err != nil {
        return err
    }

    return nil
}


func (s *Storage) Delete() {}

func (s *Storage) DeleteAll() {}
