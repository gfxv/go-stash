package core

import (
    "os"
    "path/filepath"
)


// File represents a node in a directory tree.
type File struct {
    FullPath string    
    FileName string
    IsDir    bool
    Children []File   
}

// Create directory hierarchy.
func NewFileTree(root string) (result *File, err error) {
    nodes := make(map[string]*File)
    absRoot, err := filepath.Abs(root)
    if err != nil {
	    return
    }

    walkFunc := func(path string, info os.FileInfo, err error) error {
    	if err != nil {
    		return err
    	}   

	    nodes[path] = &File {
	    	FullPath: path,
            FileName: info.Name(),
            IsDir:    info.IsDir(),
	    	Children: make([]File, 0),
	    }

	    return nil
    }

    if err = filepath.Walk(absRoot, walkFunc); err != nil {
    	return
    }

    for path, node := range nodes {
    	parentPath := filepath.Dir(path)
    	parent, exists := nodes[parentPath]
    	if !exists { // If a parent does not exist, this is the root.
    		result = node
    	} else {
    		parent.Children = append(parent.Children, *node)
    	}
    }
    return
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
