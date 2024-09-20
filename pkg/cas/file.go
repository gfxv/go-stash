package cas

import (
	"fmt"
	"os"
	"path/filepath"
)

type File struct {
	Path string
	Data []byte
}

// NewTree creates directory hierarchy.
func NewTree(root string) ([]string, error) {
	const op = "cas.file.NewTree"

	var err error
	nodes := make([]string, 0)
	walkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		if info.IsDir() {
			return nil
		}

		nodes = append(nodes, path)
		return nil
	}
	if err = filepath.Walk(root, walkFunc); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return nodes, fmt.Errorf("%s: %w", op, err)
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
