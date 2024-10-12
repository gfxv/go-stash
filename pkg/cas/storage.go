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
	BaseDir           string
	PathFunc          TransformPathFunc
	Pack              PackFunc
	Unpack            UnpackFunc
	ReplicationFactor int // TODO: implement locally
}

type Storage struct {
	baseDir       string
	transformPath TransformPathFunc
	db            *DB

	Pack   PackFunc
	Unpack UnpackFunc
}

// NewDefaultStorage creates a new instance of Storage.
//
// This function initializes the storage by creating the specified base directory,
// setting up the database, and configuring any provided transformation functions
// for paths. If any errors occur during the directory creation or database
// initialization, an error is returned.
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

// AddNewPath adds a new key-hash record to the storage database.
//
// This method takes a key and a hash as input, and it invokes the
// Add method of the underlying database to store the association
// in the database. If an error occurs during the addition process,
// it returns the error.
func (s *Storage) AddNewPath(key string, hash string) error {
	return s.db.Add(key, []string{hash})
}

// Store saves one or more file or directory paths to disk under the specified key.
//
// This method takes a key and a variable number of paths as input. It
// creates a tree structure for each path and saves it in the storage
// using the provided key. If any errors occur during the creation
// of the tree or while saving, the method returns an error.
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

// PrepareRawFile adds a special header to the beginning of a file's content
// and returns the new content as raw bytes.
//
// This function takes a file path and the original file data as input. It
// creates a header that consists of the file path followed by a null byte
// (`\0`), and prepends this header to the original data. The resulting
// byte slice containing the header and the original data is returned.
func PrepareRawFile(path string, data []byte) []byte {
	header := fmt.Sprintf("%s\u0000", path)     // header = Path + \0
	prepared := append([]byte(header), data...) // prepared = header + Data (file content in bytes)
	return prepared
}

// Write saves the given data to a file at the specified path on disk.
//
// This method takes a file path and the data to be written as input. It creates
// a new file at the specified path and writes the provided data to it. If
// any errors occur during file creation or writing, the method returns an error.
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

// WriteFromRawData writes the provided raw data to a file after
// creating path and compressing the data.
//
// This method takes raw data as input, creates path based on data, and compresses
// the data before saving it to disk. It ensures that the target directory
// exists, creating it if necessary. If a file with the same name already
// exists, it checks if the content is different to avoid overwriting.
// The method returns the transformed path of the saved file or an error
// if any operation fails.
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

// MakePathFromHash constructs a file path from a given hash.
//
// This method takes a hash string as input and creates a file path
// by splitting the hash into two parts: the prefix and the remaining
// characters. The resulting path is constructed by joining the base
// directory with the prefix and the rest of the hash.
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

// Get retrieves files associated with the given key from the storage.
//
// This method takes a key as input and fetches the associated hash values
// from the database. It constructs file paths based on the hashes and reads
// the corresponding files from disk. If any errors occur during the database
// retrieval or file reading, the method returns an error.
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

// GetByHash retrieves the content of a file as a byte slice using the provided hash.
//
// This method takes a hash string as input, constructs the file path based
// on the hash, and reads the file's content from disk. If an error occurs
// during the file reading process, the method returns an error.
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

// RecreateTree creates a directory structure and populates it with files
// based on the provided files ([]*File).
//
// This method takes a destination path and a slice of File pointers.
// It ensures that the destination folder exists, creating it if necessary.
// For each file in the provided list, it constructs the full path and checks
// if the file already exists. If it does, it compares the existing file's
// content with the new data. If the contents differ, an error is returned.
// If the file does not exist, it creates a new file with the provided data.
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

// GetHashesByKey retrieves hash values associated with the specified key.
//
// This method takes a key as input and queries the underlying database
// to fetch the corresponding hash values. If the operation is successful,
// it returns a slice of hashes; otherwise, it returns an error.
func (s *Storage) GetHashesByKey(key string) ([]string, error) {
	return s.db.GetByKey(key)
}

// RemoveByKey deletes all files associated with the specified key.
//
// This method takes a key as input and removes all files that are linked
// to that key in the storage. It first checks if the key is empty, returning
// an error if it is. Then, it retrieves the associated hash values from
// the database and attempts to remove each file by its hash. If any
// operation fails during this process, an error is returned. Finally,
// the method removes the key entry from the database.
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

// RemoveByHash deletes the file associated with the specified hash.
//
// This method takes a hash string as input and constructs the corresponding
// file path. It first checks if the file exists; if it does not, it returns
// an error. If the file exists, it attempts to remove the file from disk.
// After removing the file, it checks if the parent directory is empty and
// removes it if necessary. If any errors occur during these operations,
// the method returns an error.
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
