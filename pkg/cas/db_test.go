package cas

import (
	"errors"
	"github.com/gfxv/go-stash/internal/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDB_Add(t *testing.T) {
	const dbPath = "mock"
	utils.CreateParent(dbPath)
	defer utils.CleanUp(dbPath)

	tests := []struct {
		name        string
		key         string
		hashes      []string
		expectedErr error
	}{
		{
			name:        "Single Hash",
			key:         "key1",
			hashes:      []string{"some_hash1"},
			expectedErr: nil,
		},
		{
			name:        "Multiple Hashes",
			key:         "key2",
			hashes:      []string{"some_hash2", "some_hash3"},
			expectedErr: nil,
		},
		{
			name:        "Empty Hash List",
			key:         "key3",
			hashes:      []string{},
			expectedErr: errors.New("cas.db.Add: empty hash list"),
		},
		{
			name:        "Empty Hash",
			key:         "key4",
			hashes:      []string{""},
			expectedErr: errors.New("cas.db.Add: empty hash"),
		},
		{
			name:        "Empty Key",
			key:         "",
			hashes:      []string{"some_hash4"},
			expectedErr: errors.New("cas.db.Add: empty key"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := NewDB("mock")
			assert.NoError(t, err)

			err = db.Add(tt.key, tt.hashes)
			if err != nil && tt.expectedErr != nil && err.Error() != tt.expectedErr.Error() {
				t.Errorf("Expected error: %v, got: %v", tt.expectedErr, err)
			}
			if err == nil && tt.expectedErr != nil {
				t.Errorf("Expected error: %v, got: nil", tt.expectedErr)
			}
			if err != nil && tt.expectedErr == nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}
