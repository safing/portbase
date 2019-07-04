package database

import (
	"fmt"
	"path/filepath"

	"github.com/safing/portbase/utils"
)

const (
	databasesSubDir = "databases"
)

var (
	rootDir string
)

// GetDatabaseRoot returns the root directory of the database.
func GetDatabaseRoot() string {
	return rootDir
}

// getLocation returns the storage location for the given name and type.
func getLocation(name, storageType string) (string, error) {
	location := filepath.Join(rootDir, databasesSubDir, name, storageType)

	// check location
	err := utils.EnsureDirectory(location, 0700)
	if err != nil {
		return "", fmt.Errorf("location (%s) invalid: %s", location, err)
	}
	return location, nil
}
