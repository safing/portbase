package utils

import (
	"fmt"
	"os"
	"runtime"
)

// EnsureDirectory ensures that the given directory exists and that is has the given permissions set.
// If path is a file, it is deleted and a directory created.
// If a directory is created, also all missing directories up to the required one are created with the given permissions.
func EnsureDirectory(path string, perm os.FileMode) error {
	isDir, mode, err := mayRemoveFile(path)
	if err != nil {
		return err
	}

	if !isDir {
		err = os.MkdirAll(path, perm)
		if err != nil {
			return fmt.Errorf("could not create dir %s: %w", path, err)
		}
		return nil
	}

	if mode.Perm() != perm {
		if runtime.GOOS == "windows" {
			// TODO(ppacher)
			return nil
		}
		return os.Chmod(path, perm)
	}

	return nil
}

func mayRemoveFile(path string) (isDir bool, mode os.FileMode, err error) {
	f, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		// something happened that we cannot handle
		return false, 0, fmt.Errorf("failed to access %s: %w", path, err)
	}

	if os.IsNotExist(err) {
		// it does not exist and is not a directory
		return false, 0, nil
	}

	if f.IsDir() {
		return true, f.Mode(), nil
	}

	// f is a file so we try to remove it
	if err := os.Remove(path); err != nil {
		return false, 0, fmt.Errorf("could not remove file %s to place dir: %w", path, err)
	}

	return false, 0, nil
}
