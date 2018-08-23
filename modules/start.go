package modules

import (
	"fmt"
	"os"
	"sync"

	"github.com/Safing/portbase/log"
)

// Start starts all modules in the correct order. In case of an error, it will automatically shutdown again.
func Start() error {
	modulesLock.Lock()
	defer modulesLock.Unlock()

	// parse flags
	err := parseFlags()
	if err != nil {
		fmt.Fprintf(os.Stderr, "CRITICAL ERROR: failed to parse flags: %s\n", err)
		return err
	}

	// prep modules
	err = prepareModules()
	if err != nil {
		if err != ErrCleanExit {
			fmt.Fprintf(os.Stderr, "CRITICAL ERROR: %s\n", err)
		}
		return err
	}

	// start logging
	err = log.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "CRITICAL ERROR: failed to start logging: %s\n", err)
		return err
	}

	// start modules
	log.Info("modules: initiating...")
	err = startModules()
	if err != nil {
		log.Critical(err.Error())
		Shutdown()
		return err
	}

	startComplete.Set()
	log.Infof("modules: started %d modules", len(modules))
	return nil
}

func prepareModules() error {
	for _, module := range modulesOrder {
		err := module.prep()
		if err != nil {
			if err == ErrCleanExit {
				return ErrCleanExit
			}
			return fmt.Errorf("failed to prep module %s: %s", module.Name, err)
		}
	}
	return nil
}

func checkStartStatus() (readyToStart []*Module, done bool, err error) {
	active := 0
	modulesInProgress := false

	// go through all modules
moduleLoop:
	for _, module := range modules {
		switch {
		case module.Active.IsSet():
			active++
		case module.starting:
			modulesInProgress = true
		default:
			for _, depName := range module.dependencies {
				depModule, ok := modules[depName]
				if !ok {
					return nil, false, fmt.Errorf("modules: module %s declares dependency \"%s\", but this module has not been registered", module.Name, depName)
				}
				if !depModule.Active.IsSet() {
					continue moduleLoop
				}
			}

			readyToStart = append(readyToStart, module)
		}
	}

	// detect dependency loop
	if active < len(modules) && !modulesInProgress && len(readyToStart) == 0 {
		return nil, false, fmt.Errorf("modules: dependency loop detected, cannot continue")
	}

	if active == len(modules) {
		return nil, true, nil
	}
	return readyToStart, false, nil
}

func startModules() error {
	var modulesStarting sync.WaitGroup

	reports := make(chan error, 0)
	for {
		readyToStart, done, err := checkStartStatus()
		if err != nil {
			return err
		}

		if done {
			return nil
		}

		for _, module := range readyToStart {
			modulesStarting.Add(1)
			module.starting = true
			nextModule := module // workaround go vet alert
			go func() {
				startErr := nextModule.start()
				if startErr != nil {
					reports <- fmt.Errorf("modules: could not start module %s: %s", nextModule.Name, err)
				} else {
					log.Debugf("modules: started %s", nextModule.Name)
					nextModule.Active.Set()
					reports <- nil
				}
				modulesStarting.Done()
			}()
		}

		err = <-reports
		if err != nil {
			modulesStarting.Wait()
			return err
		}

	}
}
