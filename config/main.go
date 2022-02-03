package config

import (
	"encoding/json"
	"errors"
	"flag"
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

	exportConfig bool
)

// SetDataRoot sets the data root from which the updates module derives its paths.
func SetDataRoot(root *utils.DirStructure) {
	if dataRoot == nil {
		dataRoot = root
	}
}

func init() {
	module = modules.Register("config", prep, start, nil, "database")
	module.RegisterEvent(configChangeEvent, true)

	flag.BoolVar(&exportConfig, "export-config-options", false, "export configuration registry and exit")
}

func prep() error {
	SetDataRoot(dataroot.Root())
	if dataRoot == nil {
		return errors.New("data root is not set")
	}

	if exportConfig {
		modules.SetCmdLineOperation(exportConfigCmd)
	}

	return registerBasicOptions()
}

func start() error {
	configFilePath = filepath.Join(dataRoot.Path, "config.json")

	// Load log level from log package after it started.
	err := loadLogLevel()
	if err != nil {
		return err
	}

	err = registerAsDatabase()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	err = loadConfig()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func exportConfigCmd() error {
	data, err := json.MarshalIndent(ExportOptions(), "", "  ")
	if err != nil {
		return err
	}

	_, err = os.Stdout.Write(data)
	return err
}
