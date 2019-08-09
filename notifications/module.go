package notifications

import (
	"sync"

	"github.com/safing/portbase/modules"
)

var (
	shutdownSignal = make(chan struct{})
	shutdownWg     sync.WaitGroup
)

func init() {
	modules.Register("notifications", nil, start, nil, "base", "database")
}

func start() error {
	err := registerAsDatabase()
	if err != nil {
		return err
	}

	go cleaner()
	return nil
}

func stop() error {
	close(shutdownSignal)
	shutdownWg.Wait()
	return nil
}
