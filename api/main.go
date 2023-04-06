package api

import (
	"encoding/json"
	"errors"
	"flag"
	"os"
	"time"

	"github.com/safing/portbase/modules"
)

var (
	module *modules.Module

	exportEndpoints bool
)

// API Errors.
var (
	ErrAuthenticationAlreadySet = errors.New("the authentication function has already been set")
	ErrAuthenticationImmutable  = errors.New("the authentication function can only be set before the api has started")
)

func init() {
	module = modules.Register("api", prep, start, stop, "database", "config")

	flag.BoolVar(&exportEndpoints, "export-api-endpoints", false, "export api endpoint registry and exit")
}

func prep() error {
	if exportEndpoints {
		modules.SetCmdLineOperation(exportEndpointsCmd)
	}

	if getDefaultListenAddress() == "" {
		return errors.New("no default listen address for api available")
	}

	if err := registerConfig(); err != nil {
		return err
	}

	if err := registerDebugEndpoints(); err != nil {
		return err
	}

	if err := registerConfigEndpoints(); err != nil {
		return err
	}

	if err := registerModulesEndpoints(); err != nil {
		return err
	}

	return registerMetaEndpoints()
}

func start() error {
	startServer()

	_ = updateAPIKeys(module.Ctx, nil)
	err := module.RegisterEventHook("config", "config change", "update API keys", updateAPIKeys)
	if err != nil {
		return err
	}

	// start api auth token cleaner
	if authFnSet.IsSet() {
		module.NewTask("clean api sessions", cleanSessions).Repeat(5 * time.Minute)
	}

	return registerEndpointBridgeDB()
}

func stop() error {
	return stopServer()
}

func exportEndpointsCmd() error {
	data, err := json.MarshalIndent(ExportEndpoints(), "", "  ")
	if err != nil {
		return err
	}

	_, err = os.Stdout.Write(data)
	return err
}
