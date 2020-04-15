package api

import (
	"context"
	"errors"

	"github.com/safing/portbase/modules"
)

var (
	module *modules.Module
)

// API Errors
var (
	ErrAuthenticationAlreadySet = errors.New("the authentication function has already been set")
)

func init() {
	module = modules.Register("api", prep, start, stop, "database", "config")
}

func prep() error {
	if getDefaultListenAddress() == "" {
		return errors.New("no listen address for api available")
	}
	return registerConfig()
}

func start() error {
	logFlagOverrides()
	go Serve()
	return nil
}

func stop() error {
	if server != nil {
		return server.Shutdown(context.Background())
	}
	return nil
}
