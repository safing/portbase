package modules

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/safing/portbase/log"
	"github.com/tevino/abool"
)

var (
	modulesLock sync.RWMutex
	modules     = make(map[string]*Module)

	// ErrCleanExit is returned by Start() when the program is interrupted before starting. This can happen for example, when using the "--help" flag.
	ErrCleanExit = errors.New("clean exit requested")
)

// Module represents a module.
type Module struct {
	Name string

	// lifecycle mgmt
	Prepped      *abool.AtomicBool
	Started      *abool.AtomicBool
	Stopped      *abool.AtomicBool
	inTransition *abool.AtomicBool

	// lifecycle callback functions
	prep  func() error
	start func() error
	stop  func() error

	// shutdown mgmt
	Ctx          context.Context
	cancelCtx    func()
	shutdownFlag *abool.AtomicBool

	// workers/tasks
	workerCnt    *int32
	taskCnt      *int32
	microTaskCnt *int32
	waitGroup    sync.WaitGroup

	// events
	eventHooks     map[string][]*eventHook
	eventHooksLock sync.RWMutex

	// dependency mgmt
	depNames   []string
	depModules []*Module
	depReverse []*Module
}

// ShutdownInProgress returns whether the module has started shutting down. In most cases, you should use ShuttingDown instead.
func (m *Module) ShutdownInProgress() bool {
	return m.shutdownFlag.IsSet()
}

// ShuttingDown lets you listen for the shutdown signal.
func (m *Module) ShuttingDown() <-chan struct{} {
	return m.Ctx.Done()
}

func (m *Module) shutdown() error {
	// signal shutdown
	m.shutdownFlag.Set()
	m.cancelCtx()

	// start shutdown function
	m.waitGroup.Add(1)
	stopFnError := make(chan error, 1)
	go func() {
		stopFnError <- m.runCtrlFn("stop module", m.stop)
		m.waitGroup.Done()
	}()

	// wait for workers
	done := make(chan struct{})
	go func() {
		m.waitGroup.Wait()
		close(done)
	}()

	// wait for results
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		log.Warningf(
			"%s: timed out while waiting for workers/tasks to finish: workers=%d tasks=%d microtasks=%d, continuing shutdown...",
			m.Name,
			atomic.LoadInt32(m.workerCnt),
			atomic.LoadInt32(m.taskCnt),
			atomic.LoadInt32(m.microTaskCnt),
		)
	}

	// collect error
	select {
	case err := <-stopFnError:
		return err
	default:
		log.Warningf(
			"%s: timed out while waiting for stop function to finish, continuing shutdown...",
			m.Name,
		)
		return nil
	}
}

// Register registers a new module. The control functions `prep`, `start` and `stop` are technically optional. `stop` is called _after_ all added module workers finished.
func Register(name string, prep, start, stop func() error, dependencies ...string) *Module {
	newModule := initNewModule(name, prep, start, stop, dependencies...)

	modulesLock.Lock()
	defer modulesLock.Unlock()
	modules[name] = newModule
	return newModule
}

func initNewModule(name string, prep, start, stop func() error, dependencies ...string) *Module {
	ctx, cancelCtx := context.WithCancel(context.Background())
	var workerCnt int32
	var taskCnt int32
	var microTaskCnt int32

	newModule := &Module{
		Name:         name,
		Prepped:      abool.NewBool(false),
		Started:      abool.NewBool(false),
		Stopped:      abool.NewBool(false),
		inTransition: abool.NewBool(false),
		Ctx:          ctx,
		cancelCtx:    cancelCtx,
		shutdownFlag: abool.NewBool(false),
		waitGroup:    sync.WaitGroup{},
		workerCnt:    &workerCnt,
		taskCnt:      &taskCnt,
		microTaskCnt: &microTaskCnt,
		prep:         prep,
		start:        start,
		stop:         stop,
		eventHooks:   make(map[string][]*eventHook),
		depNames:     dependencies,
	}

	return newModule
}

func initDependencies() error {
	for _, m := range modules {
		for _, depName := range m.depNames {

			// get dependency
			depModule, ok := modules[depName]
			if !ok {
				return fmt.Errorf("module %s declares dependency \"%s\", but this module has not been registered", m.Name, depName)
			}

			// link together
			m.depModules = append(m.depModules, depModule)
			depModule.depReverse = append(depModule.depReverse, m)

		}
	}

	return nil
}

// ReadyToPrep returns whether all dependencies are ready for this module to prep.
func (m *Module) ReadyToPrep() bool {
	if m.inTransition.IsSet() || m.Prepped.IsSet() {
		return false
	}

	for _, dep := range m.depModules {
		if !dep.Prepped.IsSet() {
			return false
		}
	}

	return true
}

// ReadyToStart returns whether all dependencies are ready for this module to start.
func (m *Module) ReadyToStart() bool {
	if m.inTransition.IsSet() || m.Started.IsSet() {
		return false
	}

	for _, dep := range m.depModules {
		if !dep.Started.IsSet() {
			return false
		}
	}

	return true
}

// ReadyToStop returns whether all dependencies are ready for this module to stop.
func (m *Module) ReadyToStop() bool {
	if !m.Started.IsSet() || m.inTransition.IsSet() || m.Stopped.IsSet() {
		return false
	}

	for _, revDep := range m.depReverse {
		// not ready if a reverse dependency was started, but not yet stopped
		if revDep.Started.IsSet() && !revDep.Stopped.IsSet() {
			return false
		}
	}

	return true
}
