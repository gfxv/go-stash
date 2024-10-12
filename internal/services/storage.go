package services

import (
	"github.com/gfxv/go-stash/pkg/cas"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type StorageService struct {
	storage *cas.Storage
}

func NewStorageService(storage *cas.Storage) *StorageService {
	return &StorageService{storage: storage}
}

// SaveCompressed stores compressed data in the storage and associates it
// with the specified key and content hash.
//
// This method prepares the necessary parent directories for the storage path
// derived from the provided content hash. Then it writes the compressed data
// to the determined path in the storage. After successfully writing the
// data, it also records the key and its associated content hash in the database.
// Returns nil if the operation is successful; otherwise, it returns an error indicating the cause of failure
func (s *StorageService) SaveCompressed(key string, contentHash string, data []byte) error {
	contentPath := s.storage.MakePathFromHash(contentHash)
	err := s.storage.PrepareParentFolders(contentPath)
	if err != nil {
		return status.Errorf(codes.Internal, "can't prepare parent folders: %v", err)
	}
	err = s.storage.Write(contentPath, data)
	if err != nil {
		return status.Errorf(codes.Internal, "can't store file file to storage: %v", err)
	}

	// save path to meta.db
	err = s.storage.AddNewPath(key, contentHash)
	if err != nil {
		return status.Errorf(codes.Internal, "can't store key-hash pair")
	}

	return err
}

// SaveRaw stores raw data in the storage and associates it with the specified key.
//
// This method prepares the raw file data by adding a special header
// Then writes data to the storage, obtaining a content hash in the process.
// After successfully storing the data, it records the key and its
// associated content hash in the database.
// Returns the content hash of the stored data if successful;
// otherwise, it returns an error indicating the cause of the failure
func (s *StorageService) SaveRaw(key string, file *cas.File) (string, error) {
	data := cas.PrepareRawFile(file.Path, file.Data)
	contentHash, err := s.storage.WriteFromRawData(data)
	if err != nil {
		return "", err
	}
	if err = s.storage.AddNewPath(key, contentHash); err != nil {
		return "", err
	}

	return contentHash, nil
}

// GetHashesByKey retrieves the list of hashes associated with the specified key.
//
// This method queries the underlying storage to obtain all hashes that
// are linked to the given key. If successful, it returns the list of
// hashes. If an error occurs during the retrieval process, it returns
// an error indicating the cause of the failure.
func (s *StorageService) GetHashesByKey(key string) ([]string, error) {
	hashes, err := s.storage.GetHashesByKey(key)
	if err != nil {
		return nil, err
	}
	return hashes, nil
}

// GetFileDataByHash retrieves the file data associated with the specified hash.
//
// This method fetches the data stored under the given hash. It can
// optionally decompress the data if the `needDecompression` flag is set to true.
// If the data retrieval is successful, it returns the data, either
// in compressed or decompressed form, depending
// on the flag's value. If an error occurs during retrieval or decompression,
// it returns an error indicating the cause of the failure.
func (s *StorageService) GetFileDataByHash(hash string, needDecompression bool) ([]byte, error) {
	compressed, err := s.storage.GetByHash(hash)
	if err != nil {
		return nil, err
	}

	if !needDecompression {
		return compressed, nil
	}

	return s.storage.Unpack(compressed)
}

// GetKeysByChunks retrieves a slice of distinct keys from the storage in chunks.
//
// This method queries the underlying storage to obtain a set of keys,
// starting from the specified offset. It is useful for paginated access
// to keys, allowing for efficient retrieval without loading all keys
// at once. f an error occurs during the retrieval process, it returns
// an error indicating the cause of the failure.
func (s *StorageService) GetKeysByChunks(offset int) ([]string, error) {
	return s.storage.GetKeysByChunks(offset)
}

// MakePathFromHash generates a path based on the provided hash.
//
// This method utilizes the storage's logic to create a path that
// corresponds to the given hash.
//
// See cas.Storage's method for more details
func (s *StorageService) MakePathFromHash(hash string) string {
	return s.storage.MakePathFromHash(hash)
}

// RemoveByKey deletes all data associated with the specified key.
//
// This method invokes the underlying storage's mechanism to remove
// all files and metadata linked to the given key. If an error occurs
// during the removal process, it returns an error indicating the reason for the failure.
func (s *StorageService) RemoveByKey(key string) error {
	return s.storage.RemoveByKey(key)
}
