package api

import (
	"context"
	"errors"

	"github.com/safing/portbase/modules"
)

// API Errors
var (
	ErrAuthenticationAlreadySet = errors.New("the authentication function has already been set")
)

func init() {
	modules.Register("api", prep, start, nil, "database")
}

func prep() error {
	err := checkFlags()
	if err != nil {
		return err
	}
	return registerConfig()
}

func start() error {
	go Serve()
	return nil
}

func stop() error {
	if server != nil {
		return server.Shutdown(context.Background())
	}
	return nil
}
