package cas

import (
	"errors"
	"github.com/gfxv/go-stash/internal/utils"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func sampleStorage(baseDir string) (*Storage, error) {
	opts := StorageOpts{
		BaseDir:  baseDir,
		PathFunc: DefaultTransformPathFunc,
		Pack:     ZLibPack,
		Unpack:   ZLibUnpack,
	}

	return NewDefaultStorage(opts)
}

func TestSaveDuplicate(t *testing.T) {
	const root = "stash-test"
	defer utils.CleanUp(root)

	storage, err := sampleStorage(root)
	assert.NotNil(t, storage)
	assert.NoError(t, err)

	data := []byte("some data here")

	_, err = storage.WriteFromRawData(data)
	assert.NoError(t, err)

	_, err = storage.WriteFromRawData(data)
	assert.NoError(t, err)

}

//=============//
// RemoveByKey //
//=============//

func addSamples(s *Storage, key string, data [][]byte) error {
	for _, d := range data {
		hash, err := s.WriteFromRawData(d)
		if err != nil {
			return err
		}
		if err := s.AddNewPath(key, hash); err != nil {
			return err
		}
	}
	return nil
}

func TestStorage_RemoveByKey(t *testing.T) {
	const root = "stash-test"
	utils.CreateParent(root)
	defer utils.CleanUp(root)

	tests := []struct {
		name        string
		key         string
		data        [][]byte
		beforeFunc  func(s *Storage, key string, data [][]byte) error
		expectedErr error
	}{
		{
			name:        "Key exists",
			key:         "test_key",
			data:        [][]byte{[]byte("sample_hash1"), []byte("sample_hash2"), []byte("sample_hash3")},
			beforeFunc:  addSamples,
			expectedErr: nil,
		},
		{
			name:        "Key does not exist",
			key:         "test_key_not_exist",
			data:        [][]byte{},
			beforeFunc:  nil,
			expectedErr: nil,
		},
		{
			name:        "Empty key",
			key:         "",
			data:        [][]byte{},
			beforeFunc:  nil, // it's impossible to save key-hash pair if key (or hash) is empty, so before func is nil
			expectedErr: errors.New("cas.storage.RemoveByKey: empty key"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, err := sampleStorage(root)
			assert.NotNil(t, storage)
			assert.NoError(t, err)

			if tt.beforeFunc != nil {
				err = tt.beforeFunc(storage, tt.key, tt.data)
				assert.NoError(t, err)
			}

			err = storage.RemoveByKey(tt.key)
			utils.CheckError(t, err, tt.expectedErr)

		})
	}
}

//==============//
// RemoveByHash //
//==============//

func TestRemoveByHash(t *testing.T) {
	const root = "stash-test"
	defer utils.CleanUp(root)

	storage, err := sampleStorage(root)
	assert.NotNil(t, storage)
	assert.NoError(t, err)

	hash, err := storage.WriteFromRawData([]byte("some data here"))
	assert.NoError(t, err)

	err = storage.RemoveByHash(hash)
	assert.NoError(t, err)

	if _, err = os.Stat(storage.MakePathFromHash(hash)); os.IsExist(err) {
		t.Errorf("file not removed")
	}
}

func TestRemoveByHashNonExisting(t *testing.T) {
	const root = "stash-test"
	defer utils.CleanUp(root)

	storage, err := sampleStorage(root)
	assert.NotNil(t, storage)
	assert.NoError(t, err)

	err = storage.RemoveByHash("SOME_NON_EXISTING_HASH")
	assert.Error(t, err)

}
