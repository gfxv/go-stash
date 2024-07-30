package core

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const PREFIX_LENGTH = 5

type TransformPathFunc func([]byte) (string, string)

func DefaultTransformPathFunc(data []byte) (prefix string, filename string) {
	fullHash := sha1.Sum(data)
	strHash := hex.EncodeToString(fullHash[:])
	prefix, filename = strHash[:PREFIX_LENGTH], strHash[PREFIX_LENGTH:]
	return
}

func transformKey(key string) string {
	keyHash := sha1.Sum([]byte(key))
	return hex.EncodeToString(keyHash[:])
}

type Storage struct {
	baseDir       string
	transformPath TransformPathFunc
	db            *DB

	pack   PackFunc
	unpack UnpackFunc
}

func NewDefaultStorage(root string) *Storage {
	db, err := NewDB(root)
	if err != nil {
		panic(err) // TODO: refactor error handling
	}

	return &Storage{
		baseDir: root,
		db:      db,
	}
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
	// transformedKey := transformKey(key)
	for _, p := range paths {
		tree, err := NewTree(p)
		if err != nil {
			return err
		}
		err = s.saveTree(key, tree)
	}
	return err
}

// saveTree saves files to the disk and adds hashed paths to sqlite database
func (s *Storage) saveTree(key string, tree []string) error {
	var err error
	paths := make([]string, 0)
	for _, t := range tree {
		file, err := os.ReadFile(t)
		if err != nil {
			return err
		}
		header := fmt.Sprintf("%s\u0000", t)    // header = path + \0
		data := append([]byte(header), file...) // data = header + file content (in bytes)
		path, err := s.write(data)
		paths = append(paths, path)
	}
	err = s.db.Add(key, paths)
	return err
}

// write writes given data to the disk and returns content-based hash
func (s *Storage) write(data []byte) (string, error) {
	prefix, filename := s.transformPath(data)
	folders := fmt.Sprintf("%s/%s", s.baseDir, prefix)
	if err := os.MkdirAll(folders, os.ModePerm); err != nil { // TODO: change permissions (?), now - 777
		return "", err
	}

	fullPath := fmt.Sprintf("%s/%s", folders, filename)
	if s.Has(fullPath) {
		return "", fmt.Errorf("collision detected! \n'%s/%s' already exists", prefix, filename)
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}

	// write compressed data to storage
	compressed := s.pack(data)
	_, err = io.Copy(file, bytes.NewReader(compressed))
	if err != nil {
		return "", err
	}

	return prefix + filename, nil
}

func (s *Storage) Get(key string) ([]*File, error) {
	hashes, err := s.db.GetByKey(key)
	if err != nil {
		return nil, err
	}

	files := make([]*File, 0)
	for _, hash := range hashes {
		path := filepath.Join(s.baseDir, hash[:PREFIX_LENGTH], hash[PREFIX_LENGTH:])
		file, err := s.read(path)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

func (s *Storage) read(path string) (*File, error) {
	compressed, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	file, err := s.unpack(compressed)
	if err != nil {
		return nil, err
	}

	var originalPath string
	var originalData []byte
	for i := range file {
		if file[i] == 0 { // check if \u0000
			originalPath = string(file[:i])
			originalData = file[i+1:] // +1 to skip null-byte
			break
		}
	}

	return &File{
		path: originalPath,
		data: originalData,
	}, nil
}

// RecreateTree ...
func RecreateTree(path string, files []*File) error {
	return nil
}

func (s *Storage) Delete() {}

func (s *Storage) DeleteAll() {}
