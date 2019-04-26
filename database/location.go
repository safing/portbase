package database

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const (
	databasesSubDir = "databases"
)

var (
	rootDir string
)

func ensureDirectory(dirPath string, permissions os.FileMode) error {
	// open dir
	dir, err := os.Open(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(dirPath, permissions)
		}
		return err
	}
	defer dir.Close()

	fileInfo, err := dir.Stat()
	if err != nil {
		return err
	}
	if !fileInfo.IsDir() {
		return errors.New("path exists and is not a directory")
	}

	if runtime.GOOS == "windows" {
		// TODO
		// acl.Chmod(dirPath, permissions)
	} else if fileInfo.Mode().Perm() != permissions {
		return dir.Chmod(permissions)
	}

	return nil
}

// GetDatabaseRoot returns the root directory of the database.
func GetDatabaseRoot() string {
	return rootDir
}

// getLocation returns the storage location for the given name and type.
func getLocation(name, storageType string) (string, error) {
	location := filepath.Join(rootDir, databasesSubDir, name, storageType)

	// check location
	err := ensureDirectory(location, 0700)
	if err != nil {
		return "", fmt.Errorf("location (%s) invalid: %s", location, err)
	}
	return location, nil
}
