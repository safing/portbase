package modules

import (
	"errors"
	"fmt"

	"github.com/tevino/abool"

	"github.com/Safing/portbase/log"
)

var (
	shutdownSignal       = make(chan struct{})
	shutdownSignalClosed = abool.NewBool(false)
)

// ShuttingDown returns a channel read on the global shutdown signal.
func ShuttingDown() <-chan struct{} {
	return shutdownSignal
}

// Shutdown stops all modules in the correct order.
func Shutdown() error {

	if shutdownSignalClosed.SetToIf(false, true) {
		close(shutdownSignal)
	} else {
		// shutdown was already issued
		return errors.New("shutdown already initiated")
	}

	if startComplete.IsSet() {
		log.Warning("modules: starting shutdown...")
		modulesLock.Lock()
		defer modulesLock.Unlock()
	} else {
		log.Warning("modules: aborting, shutting down...")
	}

	err := stopModules()
	if err != nil {
		log.Error(err.Error())
		return err
	}

	log.Info("modules: shutdown complete")
	log.Shutdown()
	return nil
}

func stopModules() error {
	var rep *report
	reports := make(chan *report)
	execCnt := 0
	reportCnt := 0

	// get number of started modules
	startedCnt := 0
	for _, m := range modules {
		if m.Started.IsSet() {
			startedCnt++
		}
	}

	for {
		// find modules to exec
		for _, m := range modules {
			if m.ReadyToStop() {
				execCnt++
				m.inTransition.Set()

				execM := m
				go func() {
					reports <- &report{
						module: execM,
						err:    execM.stop(),
					}
				}()
			}
		}

		// check for dep loop
		if execCnt == reportCnt {
			return fmt.Errorf("modules: dependency loop detected, cannot continue")
		}

		// wait for reports
		rep = <-reports
		rep.module.inTransition.UnSet()
		if rep.err != nil {
			return fmt.Errorf("modules: could not stop module %s: %s", rep.module.Name, rep.err)
		}
		reportCnt++
		rep.module.Stopped.Set()
		log.Infof("modules: stopped %s", rep.module.Name)

		// exit if done
		if reportCnt == startedCnt {
			return nil
		}

	}
}
