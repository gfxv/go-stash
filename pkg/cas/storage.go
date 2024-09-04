package cas

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

func NewDefaultStorage(root string) (*Storage, error) {
	db, err := NewDB(root)
	if err != nil {
		fmt.Println("storage.NewDefaultStorage")
		return nil, err
	}

	return &Storage{
		baseDir: root,
		db:      db,
	}, nil
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

// AddNewPath adds
func (s *Storage) AddNewPath(key string, hash string) error {
	return s.db.Add(key, []string{hash})
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
		data := PrepareRawFile(t, file)
		path, err := s.WriteFromRawData(data)
		paths = append(paths, path)
	}
	err = s.db.Add(key, paths)
	return err
}

// PrepareRawFile adds a special header to the beginning of a file
// and returns new file's content as raw bytes
func PrepareRawFile(path string, data []byte) []byte {
	header := fmt.Sprintf("%s\u0000", path)     // header = Path + \0
	prepared := append([]byte(header), data...) // prepared = header + Data (file content in bytes)
	return prepared
}

// Write writes given Data to the disk
func (s *Storage) Write(path string, data []byte) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, bytes.NewReader(data))
	if err != nil {
		return err
	}

	return nil
}

// WriteFromRawData ...
func (s *Storage) WriteFromRawData(data []byte) (string, error) {
	prefix, filename := s.transformPath(data)
	folders := filepath.Join(s.baseDir, prefix)
	if err := os.MkdirAll(folders, os.ModePerm); err != nil { // TODO: change permissions (?), now - 777
		return "", err
	}

	fullPath := filepath.Join(folders, filename)
	if s.Has(fullPath) {
		return "", fmt.Errorf("stash: collision detected! \n'%s/%s' already exists", prefix, filename)
	}

	// Write compressed data to cas
	compressed := s.pack(data)
	err := s.Write(fullPath, compressed)
	if err != nil {
		return "", err
	}

	return prefix + filename, nil
}

func (s *Storage) MakePathFromHash(hash string) string {
	return filepath.Join(s.baseDir, hash[:PREFIX_LENGTH], hash[PREFIX_LENGTH:])
}

func (s *Storage) PrepareParentFolders(fullPath string) error {
	parents := filepath.Dir(fullPath)
	if err := os.MkdirAll(parents, os.ModePerm); err != nil {
		return err
	}
	return nil
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
		Path: originalPath,
		Data: originalData,
	}, nil
}

// RecreateTree ...
// TODO: probably return []error (?)
func RecreateTree(path string, files []*File) error {
	// create destination folder if it does not exist
	if !PathExists(path) {
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			return err
		}
	}

	for _, file := range files {
		fullPath := filepath.Join(path, file.Path)
		if PathExists(fullPath) {
			// (?) TODO: add some flag that allows overriding files
			err := compareFileContent(fullPath, &file.Data)
			if err != nil {
				return err
			}
			continue
		}
		err := createFile(fullPath, &file.Data)
		if err != nil {
			return err
		}
	}
	return nil
}

// compareFileContent compares content (raw bytes) of two files.
// Returns error if contents are not equal, otherwise - nil
func compareFileContent(path string, data *[]byte) error {
	f, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if bytes.Equal(f, *data) {
		return nil
	}
	return fmt.Errorf("stash: file '%s' already exists and its content is different from stashed, please remove this file manualy to avoid data overriding or corruption", path)
}

// createFile ...
func createFile(path string, data *[]byte) error {
	parent := filepath.Dir(path)
	if !PathExists(parent) {
		if err := os.MkdirAll(parent, os.ModePerm); err != nil {
			return err
		}
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		tmpErr := file.Close()
		if tmpErr != nil {
			err = tmpErr
		}
	}(file)

	// TODO: add logging ?
	_, err = file.Write(*data)
	if err != nil {
		return err
	}
	return err
}

func (s *Storage) Delete() {}

func (s *Storage) DeleteAll() {}
