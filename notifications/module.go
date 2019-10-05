package notifications

import (
	"time"

	"github.com/safing/portbase/modules"
)

var (
	module *modules.Module
)

func init() {
	module = modules.Register("notifications", nil, start, nil, "base", "database")
}

func start() error {
	err := registerAsDatabase()
	if err != nil {
		return err
	}

	go module.StartServiceWorker("cleaner", 1*time.Second, cleaner)
	return nil
}
