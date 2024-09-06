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

func (s *StorageService) SaveRaw(key string, file *cas.File) error {
	data := cas.PrepareRawFile(file.Path, file.Data)
	contentHash, err := s.storage.WriteFromRawData(data)
	if err != nil {
		return err
	}
	if err = s.storage.AddNewPath(key, contentHash); err != nil {
		return err
	}

	return nil
}

func (s *StorageService) GetHashesByKey(key string) ([]string, error) {
	hashes, err := s.storage.GetHashesByKey(key)
	if err != nil {
		return nil, err
	}
	return hashes, nil
}
