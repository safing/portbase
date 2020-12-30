package api

import (
	"context"
	"errors"
	"time"

	"github.com/safing/portbase/modules"
)

var (
	module *modules.Module
)

// API Errors
var (
	ErrAuthenticationAlreadySet = errors.New("the authentication function has already been set")
	ErrAuthenticationImmutable  = errors.New("the authentication function can only be set before the api has started")
)

func init() {
	module = modules.Register("api", prep, start, stop, "database", "config")
}

func prep() error {
	if getDefaultListenAddress() == "" {
		return errors.New("no default listen address for api available")
	}

	if err := registerConfig(); err != nil {
		return err
	}

	if err := registerDebugEndpoints(); err != nil {
		return err
	}

	return registerMetaEndpoints()
}

func start() error {
	logFlagOverrides()
	go Serve()

	// start api auth token cleaner
	if authFnSet.IsSet() {
		module.NewTask("clean api auth tokens", cleanAuthTokens).Repeat(5 * time.Minute)
	}

	return nil
}

func stop() error {
	if server != nil {
		return server.Shutdown(context.Background())
	}
	return nil
}
