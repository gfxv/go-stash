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

func (s *StorageService) GetHashesByKey(key string) ([]string, error) {
	hashes, err := s.storage.GetHashesByKey(key)
	if err != nil {
		return nil, err
	}
	return hashes, nil
}

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

func (s *StorageService) GetKeysByChunks(offset int) ([]string, error) {
	return s.storage.GetKeysByChunks(offset)
}

func (s *StorageService) MakePathFromHash(hash string) string {
	return s.storage.MakePathFromHash(hash)
}

func (s *StorageService) RemoveByKey(key string) error {
	return s.storage.RemoveByKey(key)
}
