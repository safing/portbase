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
	modulesLock sync.Mutex
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
	workerGroup  sync.WaitGroup
	workerCnt    *int32

	// dependency mgmt
	depNames   []string
	depModules []*Module
	depReverse []*Module
}

// AddWorkers adds workers to the worker waitgroup. This is a failsafe wrapper for sync.Waitgroup.
func (m *Module) AddWorkers(n uint) {
	if !m.ShutdownInProgress() {
		if atomic.AddInt32(m.workerCnt, int32(n)) > 0 {
			// only add to workgroup if cnt is positive (try to compensate wrong usage)
			m.workerGroup.Add(int(n))
		}
	}
}

// FinishWorker removes a worker from the worker waitgroup. This is a failsafe wrapper for sync.Waitgroup.
func (m *Module) FinishWorker() {
	// check worker cnt
	if atomic.AddInt32(m.workerCnt, -1) < 0 {
		log.Warningf("modules: %s module tried to finish more workers than added, this may lead to undefined behavior when shutting down", m.Name)
		return
	}
	// also mark worker done in workgroup
	m.workerGroup.Done()
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

	// wait for workers
	done := make(chan struct{})
	go func() {
		m.workerGroup.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		return errors.New("timed out while waiting for module workers to finish")
	}

	// call shutdown function
	return m.stop()
}

func dummyAction() error {
	return nil
}

// Register registers a new module. The control functions `prep`, `start` and `stop` are technically optional. `stop` is called _after_ all added module workers finished.
func Register(name string, prep, start, stop func() error, dependencies ...string) *Module {
	ctx, cancelCtx := context.WithCancel(context.Background())
	var workerCnt int32

	newModule := &Module{
		Name:         name,
		Prepped:      abool.NewBool(false),
		Started:      abool.NewBool(false),
		Stopped:      abool.NewBool(false),
		inTransition: abool.NewBool(false),
		Ctx:          ctx,
		cancelCtx:    cancelCtx,
		shutdownFlag: abool.NewBool(false),
		workerGroup:  sync.WaitGroup{},
		workerCnt:    &workerCnt,
		prep:         prep,
		start:        start,
		stop:         stop,
		depNames:     dependencies,
	}

	// replace nil arguments with dummy action
	if newModule.prep == nil {
		newModule.prep = dummyAction
	}
	if newModule.start == nil {
		newModule.start = dummyAction
	}
	if newModule.stop == nil {
		newModule.stop = dummyAction
	}

	modulesLock.Lock()
	defer modulesLock.Unlock()
	modules[name] = newModule
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
