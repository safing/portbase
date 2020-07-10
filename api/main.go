package api

import (
	"context"
	"errors"
	"time"

	"github.com/safing/portbase/modules"
)

var module *modules.Module

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
		return errors.New("no listen address for api available")
	}
	return registerConfig()
}

func start() error {
	logFlagOverrides()
	go Serve()

	// start api auth token cleaner
	authFnLock.Lock()
	defer authFnLock.Unlock()
	if authFn != nil {
		module.NewTask("clean api auth tokens", cleanAuthTokens).Repeat(time.Minute)
	}

	return nil
}

func stop() error {
	if server != nil {
		return server.Shutdown(context.Background())
	}
	return nil
}
