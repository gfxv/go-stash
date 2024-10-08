package utils

// Utils for tests

import (
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"testing"
)

func CreateParent(path string) {
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		log.Fatal(err)
	}
}

func CleanUp(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return
	}
	err := os.RemoveAll(path)
	if err != nil {
		log.Fatal(err)
	}
}

func CheckError(t *testing.T, err error, expected error) {
	if err != nil && expected != nil && err.Error() != expected.Error() {
		assert.ErrorIs(t, err, expected)
	}
	if err == nil && expected != nil {
		assert.NotNil(t, err)
	}
	if err != nil && expected == nil {
		t.Errorf("Expected no error, got: %v", err)
		assert.NoError(t, err)
	}
}
