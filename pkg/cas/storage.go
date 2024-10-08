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

type StorageOpts struct {
	BaseDir  string
	PathFunc TransformPathFunc
	Pack     PackFunc
	Unpack   UnpackFunc
}

type Storage struct {
	baseDir       string
	transformPath TransformPathFunc
	db            *DB

	Pack   PackFunc
	Unpack UnpackFunc
}

func NewDefaultStorage(opts StorageOpts) (*Storage, error) {
	const op = "cas.storage.NewDefaultStorage"

	if err := createBaseDir(opts.BaseDir); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	db, err := NewDB(opts.BaseDir)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{
		baseDir:       opts.BaseDir,
		transformPath: opts.PathFunc,
		db:            db,
		Pack:          opts.Pack,
		Unpack:        opts.Unpack,
	}, nil
}

func createBaseDir(path string) error {
	const op = "cas.storage.createBaseDir"

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.Mkdir(path, 0777); err != nil {
			return fmt.Errorf("%s: error occurred while creating the base directory: %w", op, err)
		}
	}
	return nil
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
	const op = "cas.storage.Store"

	var err error
	// transformedKey := transformKey(key)
	for _, p := range paths {
		tree, err := NewTree(p)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		err = s.saveTree(key, tree)
	}
	return fmt.Errorf("%s: %w", op, err)
}

// saveTree saves files to the disk and adds hashed paths to sqlite database
func (s *Storage) saveTree(key string, tree []string) error {
	const op = "cas.storage.saveTree"

	var err error
	paths := make([]string, 0)
	for _, t := range tree {
		file, err := os.ReadFile(t)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		data := PrepareRawFile(t, file)
		path, err := s.WriteFromRawData(data)
		paths = append(paths, path)
	}
	err = s.db.Add(key, paths)
	return fmt.Errorf("%s: %w", op, err)
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
	const op = "cas.storage.Write"

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer file.Close()

	_, err = io.Copy(file, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// WriteFromRawData ...
func (s *Storage) WriteFromRawData(data []byte) (string, error) {
	const op = "cas.storage.WriteFromRawData"

	prefix, filename := s.transformPath(data)
	compressed := s.Pack(data)

	folders := filepath.Join(s.baseDir, prefix)
	if err := os.MkdirAll(folders, os.ModePerm); err != nil { // TODO: change permissions (?), now - 777
		return "", fmt.Errorf("%s: %w", op, err)
	}

	fullPath := filepath.Join(folders, filename)
	// check if file with given name (hash) exists and its content is different
	if s.Has(fullPath) {
		if err := compareFileContent(fullPath, &compressed); err != nil {
			return "", fmt.Errorf("%s: %w", op, err)
		}
		//return "", fmt.Errorf("stash: collision detected! \n'%s/%s' already exists", prefix, filename)
		return prefix + filename, nil
	}

	// Write compressed data to cas

	err := s.Write(fullPath, compressed)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return prefix + filename, nil
}

func (s *Storage) MakePathFromHash(hash string) string {
	return filepath.Join(s.baseDir, hash[:PREFIX_LENGTH], hash[PREFIX_LENGTH:])
}

func (s *Storage) PrepareParentFolders(fullPath string) error {
	const op = "cas.storage.PrepareParentFolders"

	parents := filepath.Dir(fullPath)
	if err := os.MkdirAll(parents, os.ModePerm); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (s *Storage) Get(key string) ([]*File, error) {
	const op = "cas.storage.Get"

	hashes, err := s.db.GetByKey(key)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	files := make([]*File, 0)
	for _, hash := range hashes {
		path := filepath.Join(s.baseDir, hash[:PREFIX_LENGTH], hash[PREFIX_LENGTH:])
		file, err := s.read(path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		files = append(files, file)
	}
	return files, nil
}

// GetByHash returns file content in bytes
func (s *Storage) GetByHash(hash string) ([]byte, error) {
	const op = "cas.storage.GetByHash"

	path := filepath.Join(s.baseDir, hash[:PREFIX_LENGTH], hash[PREFIX_LENGTH:])
	compressed, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return compressed, nil
}

func (s *Storage) read(path string) (*File, error) {
	const op = "cas.storage.read"

	compressed, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	file, err := s.Unpack(compressed)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
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
	const op = "cas.storage.RecreateTree"

	// create destination folder if it does not exist
	if !PathExists(path) {
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	for _, file := range files {
		fullPath := filepath.Join(path, file.Path)
		if PathExists(fullPath) {
			// (?) TODO: add some flag that allows overriding files
			err := compareFileContent(fullPath, &file.Data)
			if err != nil {
				return fmt.Errorf("%s: %w", op, err)
			}
			continue
		}
		err := createFile(fullPath, &file.Data)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}
	return nil
}

// compareFileContent compares content (raw bytes) of two files.
// Returns error if contents are not equal, otherwise - nil
func compareFileContent(path string, data *[]byte) error {
	const op = "cas.storage.compareFileContent"

	f, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if bytes.Equal(f, *data) {
		return nil
	}
	return fmt.Errorf("stash: file '%s' already exists and its content is different from stashed, please remove this file manualy to avoid data overriding or corruption", path)
}

// createFile ...
func createFile(path string, data *[]byte) error {
	const op = "cas.storage.createFile"

	parent := filepath.Dir(path)
	if !PathExists(parent) {
		if err := os.MkdirAll(parent, os.ModePerm); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
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
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (s *Storage) GetHashesByKey(key string) ([]string, error) {
	return s.db.GetByKey(key)
}

// RemoveByKey removes all files that have same key
func (s *Storage) RemoveByKey(key string) error {
	const op = "cas.storage.RemoveByKey"

	if len(key) == 0 {
		return fmt.Errorf("%s: %w", op, errors.New("empty key"))
	}

	hashes, err := s.db.GetByKey(key)
	if err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}

	for _, hash := range hashes {
		err := s.RemoveByHash(hash)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	err = s.db.RemoveByKey(key)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// RemoveByHash removes file with given hash
func (s *Storage) RemoveByHash(hash string) error {
	const op = "cas.storage.RemoveByHash"

	fullPath := filepath.Join(s.baseDir, hash[:PREFIX_LENGTH], hash[PREFIX_LENGTH:])
	if !s.Has(fullPath) {
		return fmt.Errorf("%s: %w", op, os.ErrNotExist)
	}

	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	// remove parent directory if its empty
	parent := filepath.Dir(fullPath)
	dir, err := os.Open(parent)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer dir.Close()

	_, err = dir.Readdirnames(1)
	if err == io.EOF { // if directory is empty
		if err := os.Remove(parent); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	return nil
}

func (s *Storage) DeleteAll() {}

func (s *Storage) GetKeysByChunks(offset int) ([]string, error) {
	return s.db.GetKeysByChunks(offset)
}
