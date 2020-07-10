package api

import (
	"context"
	"time"

	"github.com/safing/portbase/modules"
)

var module *modules.Module

func init() {
	module = modules.Register("api", prep, start, stop, "database", "config")
}

func prep() error {
	if getDefaultListenAddress() == "" {
		return errNoListenAddr
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
