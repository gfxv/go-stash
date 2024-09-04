package cas

import (
	"os"
	"path/filepath"
)

type File struct {
	Path string
	Data []byte
}

// NewTree creates directory hierarchy.
func NewTree(root string) ([]string, error) {
	var err error
	nodes := make([]string, 0)
	walkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		nodes = append(nodes, path)
		return nil
	}
	if err = filepath.Walk(root, walkFunc); err != nil {
		return nil, err
	}

	return nodes, err
}

// IsDir reports whether path describes a directory.
// Returns err if path does not exist
func isDir(path string) (bool, error) {
	f, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return f.IsDir(), nil
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	return os.IsExist(err)
}
