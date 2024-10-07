package cas

import (
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

func TestRemoveByKey(t *testing.T) {
	const root = "stash-test"
	defer utils.CleanUp(root)

	storage, err := sampleStorage(root)
	assert.NotNil(t, storage)
	assert.NoError(t, err)

	const key = "test_key"

	hash, err := storage.WriteFromRawData([]byte("some data 1"))
	assert.NoError(t, err)
	err = storage.AddNewPath(key, hash)
	assert.NoError(t, err)

	hash, err = storage.WriteFromRawData([]byte("some data 2"))
	assert.NoError(t, err)
	err = storage.AddNewPath(key, hash)
	assert.NoError(t, err)

	err = storage.RemoveByKey(key)
	assert.NoError(t, err)

	hashes, err := storage.GetHashesByKey(key)
	assert.NoError(t, err)
	assert.Len(t, hashes, 0)
}

func TestRemoveByKeyNonExisting(t *testing.T) {
	const root = "stash-test"
	defer utils.CleanUp(root)

	storage, err := sampleStorage(root)
	assert.NotNil(t, storage)
	assert.NoError(t, err)

	const key = "test_key"
	err = storage.RemoveByKey(key)
	assert.NoError(t, err)
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
