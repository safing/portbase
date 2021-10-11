package modules

import (
	"errors"
	"fmt"

	"github.com/tevino/abool"

	"github.com/safing/portbase/log"
)

var (
	shutdownSignal = make(chan struct{})
	shutdownFlag   = abool.NewBool(false)

	shutdownCompleteSignal = make(chan struct{})

	globalShutdownFn func()
)

// SetGlobalShutdownFn sets a global shutdown function that is called first when shutting down.
func SetGlobalShutdownFn(fn func()) {
	if globalShutdownFn == nil {
		globalShutdownFn = fn
	}
}

// IsShuttingDown returns whether the global shutdown is in progress.
func IsShuttingDown() bool {
	return shutdownFlag.IsSet()
}

// ShuttingDown returns a channel read on the global shutdown signal.
func ShuttingDown() <-chan struct{} {
	return shutdownSignal
}

// Shutdown stops all modules in the correct order.
func Shutdown() error {
	// lock mgmt
	mgmtLock.Lock()
	defer mgmtLock.Unlock()

	if shutdownFlag.SetToIf(false, true) {
		close(shutdownSignal)
	} else {
		// shutdown was already issued
		return errors.New("shutdown already initiated")
	}

	// Execute global shutdown function.
	if globalShutdownFn != nil {
		globalShutdownFn()
	}

	if initialStartCompleted.IsSet() {
		log.Warning("modules: starting shutdown...")
	} else {
		log.Warning("modules: aborting, shutting down...")
	}

	err := stopModules()
	if err != nil {
		log.Errorf("modules: shutdown completed with error: %s", err)
	} else {
		log.Info("modules: shutdown completed")
	}

	log.Shutdown()
	close(shutdownCompleteSignal)
	return err
}

func stopModules() error {
	var rep *report
	var lastErr error
	reports := make(chan *report)
	execCnt := 0
	reportCnt := 0

	// get number of started modules
	startedCnt := 0
	for _, m := range modules {
		if m.Status() >= StatusStarting {
			startedCnt++
		}
	}

	for {
		waiting := 0

		// find modules to exec
		for _, m := range modules {
			switch m.readyToStop() {
			case statusNothingToDo:
			case statusWaiting:
				waiting++
			case statusReady:
				execCnt++
				m.stop(reports)
			}
		}

		if reportCnt < execCnt {
			// wait for reports
			rep = <-reports
			if rep.err != nil {
				lastErr = rep.err
				log.Warningf("modules: could not stop module %s: %s", rep.module.Name, rep.err)
			}
			reportCnt++
			log.Infof("modules: stopped %s", rep.module.Name)
		} else {
			// finished
			if waiting > 0 {
				// check for dep loop
				return fmt.Errorf("modules: dependency loop detected, cannot continue")
			}
			// return last error
			return lastErr
		}
	}
}
