package modules

import (
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/tevino/abool"

	"github.com/safing/portbase/log"
)

// ErrModuleSystemStarted indicates that the module system has already
// been started.
var ErrModuleSystemStarted = errors.New("module system already started")

var (
	initialStartCompleted = abool.NewBool(false)
	globalPrepFn          func() error
)

// SetGlobalPrepFn sets a global prep function that is run before
// all modules. This can be used to pre-initialize modules, such as
// setting the data root or database path.
func SetGlobalPrepFn(fn func() error) {
	if globalPrepFn == nil {
		globalPrepFn = fn
	}
}

// Start starts all modules in the correct order. In case of an error,
// it will automatically shutdown again.
func Start() error {
	if !modulesLocked.SetToIf(false, true) {
		return ErrModuleSystemStarted
	}

	// lock mgmt
	mgmtLock.Lock()
	defer mgmtLock.Unlock()

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
		if err != ErrCleanExit {
			fmt.Fprintf(os.Stderr, "CRITICAL ERROR: failed to parse flags: %s\n", err)
		}
		return err
	}

	// execute global prep fn
	if globalPrepFn != nil {
		err = globalPrepFn()
		if err != nil {
			if err != ErrCleanExit {
				fmt.Fprintf(os.Stderr, "CRITICAL ERROR: %s\n", err)
			}
			return err
		}
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

	// build dependency tree
	buildEnabledTree()

	// start modules
	log.Info("modules: initiating...")
	err = startModules()
	if err != nil {
		log.Critical(err.Error())
		return err
	}

	// complete startup
	if moduleMgmtEnabled.IsSet() {
		log.Info("modules: initiated subsystems manager")
	} else {
		log.Infof("modules: started %d modules", len(modules))
	}

	go taskQueueHandler()
	go taskScheduleHandler()

	initialStartCompleted.Set()
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
		waiting := 0

		// find modules to exec
		for _, m := range modules {
			switch m.readyToPrep() {
			case statusNothingToDo:
			case statusWaiting:
				waiting++
			case statusReady:
				execCnt++
				m.prep(reports)
			}
		}

		if reportCnt >= execCnt {
			// finished
			if waiting > 0 {
				// check for dep loop
				return fmt.Errorf("modules: dependency loop detected, cannot continue")
			}
			return nil
		}

		rep = <-reports
		if rep.err != nil {
			if rep.err == ErrCleanExit {
				return rep.err
			}
			return fmt.Errorf("failed to prep module %s: %w", rep.module.Name, rep.err)
		}
		reportCnt++

	}
}

func startModules() error {
	var rep *report
	reports := make(chan *report)
	execCnt := 0
	reportCnt := 0

	for {
		waiting := 0

		// find modules to exec
		for _, m := range modules {
			switch m.readyToStart() {
			case statusNothingToDo:
			case statusWaiting:
				waiting++
			case statusReady:
				execCnt++
				m.start(reports)
			}
		}

		if reportCnt >= execCnt {
			// finished
			if waiting > 0 {
				// check for dep loop
				return fmt.Errorf("modules: dependency loop detected, cannot continue")
			}
			// return last error
			return nil
		}

		// wait for reports
		rep = <-reports
		if rep.err != nil {
			return fmt.Errorf("modules: could not start module %s: %w", rep.module.Name, rep.err)
		}
		reportCnt++
		log.Infof("modules: started %s", rep.module.Name)
	}
}
