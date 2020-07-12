package modules

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/safing/portbase/log"
	"github.com/tevino/abool"
)

var (
	modules  = make(map[string]*Module)
	mgmtLock sync.Mutex

	// lock modules when starting
	modulesLocked = abool.New()

	moduleStartTimeout = 2 * time.Minute
	moduleStopTimeout  = 1 * time.Minute
)

// Module represents a module.
type Module struct { //nolint:maligned // not worth the effort
	sync.RWMutex

	Name string

	// status mgmt
	enabled             *abool.AtomicBool
	enabledAsDependency *abool.AtomicBool
	status              uint8

	// failure status
	failureStatus uint8
	failureID     string
	failureMsg    string

	// lifecycle callback functions
	prepFn  func() error
	startFn func() error
	stopFn  func() error

	// lifecycle mgmt
	// start
	startComplete chan struct{}
	// stop
	Ctx       context.Context
	cancelCtx func()
	stopFlag  *abool.AtomicBool

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

// StartCompleted returns a channel read that triggers when the module has finished starting.
func (m *Module) StartCompleted() <-chan struct{} {
	m.RLock()
	defer m.RUnlock()
	return m.startComplete
}

// Stopping returns a channel read that triggers when the module has initiated the stop procedure.
func (m *Module) Stopping() <-chan struct{} {
	m.RLock()
	defer m.RUnlock()
	return m.Ctx.Done()
}

// IsStopping returns whether the module has started shutting down. In most cases, you should use Stopping instead.
func (m *Module) IsStopping() bool {
	return m.stopFlag.IsSet()
}

// Dependencies returns the module's dependencies.
func (m *Module) Dependencies() []*Module {
	m.RLock()
	defer m.RUnlock()
	return m.depModules
}

func (m *Module) prep(reports chan *report) {
	// check and set intermediate status
	m.Lock()
	if m.status != StatusDead {
		m.Unlock()
		go func() {
			reports <- &report{
				module: m,
				err:    ErrModulePrepared,
			}
		}()
		return
	}
	m.status = StatusPreparing
	m.Unlock()

	// run prep function
	go func() {
		var err error
		if m.prepFn != nil {
			// execute function
			err = m.runCtrlFnWithTimeout(
				"prep module",
				moduleStartTimeout,
				m.prepFn,
			)
		}
		// set status
		if err != nil {
			m.Error(
				"module-failed-prep",
				fmt.Sprintf("failed to prep module: %s", err.Error()),
			)
		} else {
			m.Lock()
			m.status = StatusOffline
			m.Unlock()
			m.notifyOfChange()
		}
		// send report
		reports <- &report{
			module: m,
			err:    err,
		}
	}()
}

func (m *Module) start(reports chan *report) {
	// check and set intermediate status
	m.Lock()
	if m.status != StatusOffline {
		m.Unlock()
		go func() {
			reports <- &report{
				module: m,
				err:    ErrModuleNotOffline,
			}
		}()
		return
	}
	m.status = StatusStarting

	// reset stop management
	if m.cancelCtx != nil {
		// trigger cancel just to be sure
		m.cancelCtx()
	}
	m.Ctx, m.cancelCtx = context.WithCancel(context.Background())
	m.stopFlag.UnSet()

	m.Unlock()

	// run start function
	go func() {
		var err error
		if m.startFn != nil {
			// execute function
			err = m.runCtrlFnWithTimeout(
				"start module",
				moduleStartTimeout,
				m.startFn,
			)
		}
		// set status
		if err != nil {
			m.Error(
				"module-failed-start",
				fmt.Sprintf("failed to start module: %s", err.Error()),
			)
		} else {
			m.Lock()
			m.status = StatusOnline
			// init start management
			close(m.startComplete)
			m.Unlock()
			m.notifyOfChange()
		}
		// send report
		reports <- &report{
			module: m,
			err:    err,
		}
	}()
}

func (m *Module) stop(reports chan *report) {
	// check and set intermediate status
	m.Lock()
	if m.status != StatusOnline {
		m.Unlock()
		go func() {
			reports <- &report{
				module: m,
				err:    ErrModuleNotOnline,
			}
		}()
		return
	}
	m.status = StatusStopping

	// reset start management
	m.startComplete = make(chan struct{})
	// init stop management
	m.cancelCtx()
	m.stopFlag.Set()

	m.Unlock()

	go m.stopAllTasks(reports)
}

func (m *Module) stopAllTasks(reports chan *report) {
	// start shutdown function
	stopFnFinished := abool.NewBool(false)
	var stopFnError error
	if m.stopFn != nil {
		m.waitGroup.Add(1)
		go func() {
			stopFnError = m.runCtrlFn("stop module", m.stopFn)
			stopFnFinished.Set()
			m.waitGroup.Done()
		}()
	}

	// wait for workers and stop fn
	done := make(chan struct{})
	go func() {
		m.waitGroup.Wait()
		close(done)
	}()

	// wait for results
	select {
	case <-done:
	case <-time.After(moduleStopTimeout):
		log.Warningf(
			"%s: timed out while waiting for stopfn/workers/tasks to finish: stopFn=%v workers=%d tasks=%d microtasks=%d, continuing shutdown...",
			m.Name,
			stopFnFinished.IsSet(),
			atomic.LoadInt32(m.workerCnt),
			atomic.LoadInt32(m.taskCnt),
			atomic.LoadInt32(m.microTaskCnt),
		)
	}

	// collect error
	var err error
	if stopFnFinished.IsSet() && stopFnError != nil {
		err = stopFnError
	}
	// set status
	if err != nil {
		m.Error(
			"module-failed-stop",
			fmt.Sprintf("failed to stop module: %s", err.Error()),
		)
	} else {
		m.Lock()
		m.status = StatusOffline
		m.Unlock()
		m.notifyOfChange()
	}
	// send report
	reports <- &report{
		module: m,
		err:    err,
	}
}

// Register registers a new module. The control functions `prep`, `start` and `stop` are technically optional. `stop` is called _after_ all added module workers finished.
func Register(name string, prep, start, stop func() error, dependencies ...string) *Module {
	if modulesLocked.IsSet() {
		return nil
	}

	newModule := initNewModule(name, prep, start, stop, dependencies...)

	// check for already existing module
	_, ok := modules[name]
	if ok {
		panic(fmt.Sprintf("modules: module %s is already registered", name))
	}
	// add new module
	modules[name] = newModule

	return newModule
}

func initNewModule(name string, prep, start, stop func() error, dependencies ...string) *Module {
	ctx, cancelCtx := context.WithCancel(context.Background())
	var workerCnt int32
	var taskCnt int32
	var microTaskCnt int32

	newModule := &Module{
		Name:                name,
		enabled:             abool.NewBool(false),
		enabledAsDependency: abool.NewBool(false),
		prepFn:              prep,
		startFn:             start,
		stopFn:              stop,
		startComplete:       make(chan struct{}),
		Ctx:                 ctx,
		cancelCtx:           cancelCtx,
		stopFlag:            abool.NewBool(false),
		workerCnt:           &workerCnt,
		taskCnt:             &taskCnt,
		microTaskCnt:        &microTaskCnt,
		waitGroup:           sync.WaitGroup{},
		eventHooks:          make(map[string][]*eventHook),
		depNames:            dependencies,
	}

	return newModule
}

func initDependencies() error {
	for _, m := range modules {
		for _, depName := range m.depNames {

			// get dependency
			depModule, ok := modules[depName]
			if !ok {
				return fmt.Errorf("dependecy %s of %s: %w", depName, m.Name, ErrUnknownModule)
			}

			// link together
			m.depModules = append(m.depModules, depModule)
			depModule.depReverse = append(depModule.depReverse, m)

		}
	}

	return nil
}
