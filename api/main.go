package api

import (
	"github.com/safing/portbase/modules"
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
