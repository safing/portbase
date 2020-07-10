package config

import (
	"os"
	"path/filepath"

	"github.com/safing/portbase/dataroot"
	"github.com/safing/portbase/modules"
	"github.com/safing/portbase/utils"
)

const (
	configChangeEvent = "config change"
)

var (
	module   *modules.Module
	dataRoot *utils.DirStructure
)

// SetDataRoot sets the data root from which the updates module derives its paths.
func SetDataRoot(root *utils.DirStructure) {
	if dataRoot == nil {
		dataRoot = root
	}
}

func init() {
	module = modules.Register("config", prep, start, nil, "database")
	module.RegisterEvent(configChangeEvent)
}

func prep() error {
	SetDataRoot(dataroot.Root())
	if dataRoot == nil {
		return dataroot.ErrNotSet
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
