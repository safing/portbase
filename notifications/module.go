package notifications

import (
	"time"

	"github.com/safing/portbase/modules"
)

var (
	module *modules.Module
)

func init() {
	module = modules.Register("notifications", prep, start, nil, "database", "base")
}

func prep() error {
	return registerConfig()
}

func start() error {
	err := registerAsDatabase()
	if err != nil {
		return err
	}

	go module.StartServiceWorker("cleaner", 1*time.Second, cleaner)
	return nil
}
