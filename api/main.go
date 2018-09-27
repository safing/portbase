package api

import (
	"github.com/Safing/portbase/modules"
)

func init() {
	modules.Register("api", prep, start, stop, "database")
}

func prep() error {
	return nil
}

func start() error {
	go Serve()
	return nil
}

func stop() error {
	return nil
}
