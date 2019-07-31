package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/safing/portbase/modules"
	"github.com/safing/portbase/utils"
	"github.com/safing/portmaster/core/structure"
)

var (
	dataRoot *utils.DirStructure
)

// SetDataRoot sets the data root from which the updates module derives its paths.
func SetDataRoot(root *utils.DirStructure) {
	if dataRoot == nil {
		dataRoot = root
	}
}

func init() {
	modules.Register("config", prep, start, nil, "core")
}

func prep() error {
	SetDataRoot(structure.Root())
	if dataRoot == nil {
		return errors.New("data root is not set")
	}
	return nil
}

func start() error {
	configFilePath = filepath.Join(dataRoot.Path, "config.json")

	err := registerAsDatabase()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	err = loadConfig()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
