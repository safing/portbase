package modules

import (
	"fmt"

	"github.com/tevino/abool"

	"github.com/Safing/portbase/log"
)

var (
	shutdownSignal = make(chan struct{})
	shutdownSignalClosed = abool.NewBool(false)
)

// ShuttingDown returns a channel read on the global shutdown signal.
func ShuttingDown() <-chan struct{} {
	return shutdownSignal
}

func checkStopStatus() (readyToStop []*Module, done bool) {
	active := 0

	// collect all active modules
	activeModules := make(map[string]*Module)
	for _, module := range modules {
		if module.Active.IsSet() {
			active++
			activeModules[module.Name] = module
		}
	}
	if active == 0 {
		return nil, true
	}

	// remove modules that others depend on
	for _, module := range activeModules {
		for _, depName := range module.dependencies {
			delete(activeModules, depName)
		}
	}

	// make list out of map
	for _, module := range activeModules {
		readyToStop = append(readyToStop, module)
	}

	return readyToStop, false
}

// Shutdown stops all modules in the correct order.
func Shutdown() error {

	if startComplete.IsSet() {
		log.Warning("modules: starting shutdown...")
		modulesLock.Lock()
		defer modulesLock.Unlock()
	} else {
		log.Warning("modules: aborting, shutting down...")
	}

	if shutdownSignalClosed.SetToIf(false, true) {
		close(shutdownSignal)
	}

	reports := make(chan error, 0)
	for {
		readyToStop, done := checkStopStatus()

		if done {
			log.Info("modules: shutdown complete")
			return nil
		}

		for _, module := range readyToStop {
			module.starting = false
			nextModule := module // workaround go vet alert
			go func() {
				err := nextModule.stop()
				nextModule.Active.UnSet()
				if err != nil {
					reports <- fmt.Errorf("modules: could not stop module %s: %s", nextModule.Name, err)
				} else {
					reports <- nil
				}
			}()
		}

		err := <-reports
		if err != nil {
			log.Error(err.Error())
			return err
		}

	}
}
