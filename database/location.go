package database

import (
	"errors"
	"fmt"
	"os"
	"path"
)

const (
	databasesSubDir = "databases"
)

var (
	rootDir string
)

func ensureDirectory(dirPath string) error {
	// open dir
	dir, err := os.Open(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(dirPath, 0700)
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
	if fileInfo.Mode().Perm() != 0700 {
		return dir.Chmod(0700)
	}
	return nil
}

// GetDatabaseRoot returns the root directory of the database.
func GetDatabaseRoot() string {
	return rootDir
}

// getLocation returns the storage location for the given name and type.
func getLocation(name, storageType string) (string, error) {
	location := path.Join(rootDir, databasesSubDir, name, storageType)

	// check location
	err := ensureDirectory(location)
	if err != nil {
		return "", fmt.Errorf("location (%s) invalid: %s", location, err)
	}
	return location, nil
}
