package database

import (
	"path"
	"os"
	"fmt"
	"errors"
)

var (
	rootDir string
)

// Initialize initialized the database
func Initialize(location string) error {
	if initialized.SetToIf(false, true) {
		rootDir = location

		err := checkRootDir()
		if err != nil {
			return fmt.Errorf("could not create/open database directory (%s): %s", rootDir, err)
		}

		err = loadRegistry()
		if err != nil {
			return fmt.Errorf("could not load database registry (%s): %s", path.Join(rootDir, registryFileName), err)
		}

		return nil
	}
	return errors.New("database already initialized")
}

func checkRootDir() error {
	// open dir
	dir, err := os.Open(rootDir)
	if err != nil {
		if err == os.ErrNotExist {
			return os.MkdirAll(rootDir, 0700)
		}
		return err
	}
	defer dir.Close()

	fileInfo, err := dir.Stat()
	if err != nil {
		return err
	}

	if fileInfo.Mode().Perm() != 0700 {
		return dir.Chmod(0700)
	}
	return nil
}

// getLocation returns the storage location for the given name and type.
func getLocation(name, storageType string) (location string, err error) {
	return path.Join(rootDir, name, storageType), nil
}
