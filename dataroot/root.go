package dataroot

import (
	"errors"
	"os"

	"github.com/safing/portbase/utils"
)

// Common errors.
var (
	ErrAlreadyInitialized = errors.New("already initialized")
	ErrNotSet             = errors.New("data root is not set")
)

var root *utils.DirStructure

// Initialize initializes the data root directory.
func Initialize(rootDir string, perm os.FileMode) error {
	if root != nil {
		return ErrAlreadyInitialized
	}

	root = utils.NewDirStructure(rootDir, perm)
	return root.Ensure()
}

// Root returns the data root directory.
func Root() *utils.DirStructure {
	return root
}
