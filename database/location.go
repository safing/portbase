package database

import (
	"path"
)

var (
	rootDir string
)

// getLocation returns the storage location for the given name and type.
func getLocation(name, storageType string) (location string, err error) {
	return path.Join(rootDir, name, storageType), nil
}
