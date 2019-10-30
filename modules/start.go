package modules

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/safing/portbase/log"
	"github.com/tevino/abool"
)

var (
	startComplete       = abool.NewBool(false)
	startCompleteSignal = make(chan struct{})
)

// StartCompleted returns whether starting has completed.
func StartCompleted() bool {
	return startComplete.IsSet()
}

// WaitForStartCompletion returns as soon as starting has completed.
func WaitForStartCompletion() <-chan struct{} {
	return startCompleteSignal
}

// Start starts all modules in the correct order. In case of an error, it will automatically shutdown again.
func Start() error {
	modulesLock.RLock()
	defer modulesLock.RUnlock()

	// start microtask scheduler
	go microTaskScheduler()
	SetMaxConcurrentMicroTasks(runtime.GOMAXPROCS(0) * 2)

	// inter-link modules
	err := initDependencies()
	if err != nil {
		fmt.Fprintf(os.Stderr, "CRITICAL ERROR: failed to initialize modules: %s\n", err)
		return err
	}

	// parse flags
	err = parseFlags()
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
	log.EnableScheduling()
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
		return err
	}

	// complete startup
	log.Infof("modules: started %d modules", len(modules))
	if startComplete.SetToIf(false, true) {
		close(startCompleteSignal)
	}

	go taskQueueHandler()
	go taskScheduleHandler()

	return nil
}

type report struct {
	module *Module
	err    error
}

func prepareModules() error {
	var rep *report
	reports := make(chan *report)
	execCnt := 0
	reportCnt := 0

	for {
		// find modules to exec
		for _, m := range modules {
			if m.ReadyToPrep() {
				execCnt++
				m.inTransition.Set()

				execM := m
				go func() {
					reports <- &report{
						module: execM,
						err: execM.runCtrlFnWithTimeout(
							"prep module",
							10*time.Second,
							execM.prep,
						),
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
			if rep.err == ErrCleanExit {
				return rep.err
			}
			return fmt.Errorf("failed to prep module %s: %s", rep.module.Name, rep.err)
		}
		reportCnt++
		rep.module.Prepped.Set()

		// exit if done
		if reportCnt == len(modules) {
			return nil
		}

	}
}

func startModules() error {
	var rep *report
	reports := make(chan *report)
	execCnt := 0
	reportCnt := 0

	for {
		// find modules to exec
		for _, m := range modules {
			if m.ReadyToStart() {
				execCnt++
				m.inTransition.Set()

				execM := m
				go func() {
					reports <- &report{
						module: execM,
						err: execM.runCtrlFnWithTimeout(
							"start module",
							60*time.Second,
							execM.start,
						),
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
			return fmt.Errorf("modules: could not start module %s: %s", rep.module.Name, rep.err)
		}
		reportCnt++
		rep.module.Started.Set()
		log.Infof("modules: started %s", rep.module.Name)

		// exit if done
		if reportCnt == len(modules) {
			return nil
		}

	}
}
