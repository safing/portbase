package modules

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tevino/abool"

	"github.com/safing/portbase/log"
)

var (
	modules  = make(map[string]*Module)
	mgmtLock sync.Mutex

	// modulesLocked locks `modules` during starting.
	modulesLocked = abool.New()

	sleepMode                         = abool.NewBool(false)
	taskSchedulerSleepModeExitChannel = make(chan struct{})

	moduleStartTimeout = 2 * time.Minute
	moduleStopTimeout  = 1 * time.Minute

	// ErrCleanExit is returned by Start() when the program is interrupted before starting. This can happen for example, when using the "--help" flag.
	ErrCleanExit = errors.New("clean exit requested")
)

// Module represents a module.
type Module struct {
	sync.RWMutex

	Name string

	// status mgmt
	enabled             *abool.AtomicBool
	enabledAsDependency *abool.AtomicBool
	status              uint8
	sleepMode           *abool.AtomicBool
	sleepWaitingChannel chan time.Time

	// failure status
	failureStatus uint8
	failureID     string
	failureTitle  string
	failureMsg    string

	// lifecycle callback functions
	prepFn  func() error
	startFn func() error
	stopFn  func() error

	// lifecycle mgmt
	// start
	startComplete chan struct{}
	// stop
	Ctx          context.Context
	cancelCtx    func()
	stopFlag     *abool.AtomicBool
	stopComplete chan struct{}

	// workers/tasks
	ctrlFuncRunning *abool.AtomicBool
	workerCnt       *int32
	taskCnt         *int32
	microTaskCnt    *int32

	// events
	eventHooks     map[string]*eventHooks
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

// Sleep enables or disables sleep mode.
func (m *Module) Sleep(enable bool) {
	set := m.sleepMode.SetToIf(!enable, enable)
	if !set {
		return
	}

	// Notify all waiting tasks that we are not sleeping anymore.
	m.Lock()
	defer m.Unlock()

	if enable {
		m.sleepWaitingChannel = make(chan time.Time)
	} else {
		close(m.sleepWaitingChannel)
	}
}

// IsSleeping returns true if sleep mode is enabled.
func (m *Module) IsSleeping() bool {
	return m.sleepMode.IsSet()
}

// WaitIfSleeping returns channel that will signal when it exits sleep mode.
func (m *Module) WaitIfSleeping() <-chan time.Time {
	m.RLock()
	defer m.RUnlock()
	return m.sleepWaitingChannel
}

// NewSleepyTicker returns new sleepyTicker that will respect the modules sleep mode.
func (m *Module) NewSleepyTicker(normalDuration time.Duration, sleepDuration time.Duration) *SleepyTicker {
	return newSleepyTicker(m, normalDuration, sleepDuration)
}

func (m *Module) prep(reports chan *report) {
	// check and set intermediate status
	m.Lock()
	if m.status != StatusDead {
		m.Unlock()
		go func() {
			reports <- &report{
				module: m,
				err:    fmt.Errorf("module already prepped"),
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
				fmt.Sprintf("%s:prep-failed", m.Name),
				fmt.Sprintf("Preparing module %s failed", m.Name),
				fmt.Sprintf("Failed to prep module: %s", err.Error()),
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
				err:    fmt.Errorf("module not offline"),
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
				fmt.Sprintf("%s:start-failed", m.Name),
				fmt.Sprintf("Starting module %s failed", m.Name),
				fmt.Sprintf("Failed to start module: %s", err.Error()),
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

func (m *Module) checkIfStopComplete() {
	if m.stopFlag.IsSet() &&
		m.ctrlFuncRunning.IsNotSet() &&
		atomic.LoadInt32(m.workerCnt) == 0 &&
		atomic.LoadInt32(m.taskCnt) == 0 &&
		atomic.LoadInt32(m.microTaskCnt) == 0 {

		m.Lock()
		defer m.Unlock()

		if m.stopComplete != nil {
			close(m.stopComplete)
			m.stopComplete = nil
		}
	}
}

func (m *Module) stop(reports chan *report) {
	m.Lock()
	defer m.Unlock()

	// check and set intermediate status
	if m.status != StatusOnline {
		go func() {
			reports <- &report{
				module: m,
				err:    fmt.Errorf("module not online"),
			}
		}()
		return
	}

	// Reset start/stop signal channels.
	m.startComplete = make(chan struct{})
	m.stopComplete = make(chan struct{})

	// Make a copy of the stop channel.
	stopComplete := m.stopComplete

	// Set status and cancel context.
	m.status = StatusStopping
	m.stopFlag.Set()
	m.cancelCtx()

	go m.stopAllTasks(reports, stopComplete)
}

func (m *Module) stopAllTasks(reports chan *report, stopComplete chan struct{}) {
	// start shutdown function
	var stopFnError error
	stopFuncRunning := abool.New()
	if m.stopFn != nil {
		stopFuncRunning.Set()
		go func() {
			stopFnError = m.runCtrlFn("stop module", m.stopFn)
			stopFuncRunning.UnSet()
			m.checkIfStopComplete()
		}()
	} else {
		m.checkIfStopComplete()
	}

	// wait for results
	select {
	case <-stopComplete:
	// case <-time.After(moduleStopTimeout):
	case <-time.After(moduleStopTimeout):
		log.Warningf(
			"%s: timed out while waiting for stopfn/workers/tasks to finish: stopFn=%v/%v workers=%d tasks=%d microtasks=%d, continuing shutdown...",
			m.Name,
			stopFuncRunning.IsSet(), m.ctrlFuncRunning.IsSet(),
			atomic.LoadInt32(m.workerCnt),
			atomic.LoadInt32(m.taskCnt),
			atomic.LoadInt32(m.microTaskCnt),
		)
	}

	// collect error
	var err error
	if stopFuncRunning.IsNotSet() && stopFnError != nil {
		err = stopFnError
	}
	// set status
	if err != nil {
		m.Error(
			fmt.Sprintf("%s:stop-failed", m.Name),
			fmt.Sprintf("Stopping module %s failed", m.Name),
			fmt.Sprintf("Failed to stop module: %s", err.Error()),
		)
	}

	// Always set to offline in order to let other modules shutdown in order.
	m.Lock()
	m.status = StatusOffline
	m.Unlock()
	m.notifyOfChange()

	// Resolve any errors still on the module.
	m.Resolve("")

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
		sleepMode:           abool.NewBool(false),
		sleepWaitingChannel: make(chan time.Time),
		prepFn:              prep,
		startFn:             start,
		stopFn:              stop,
		startComplete:       make(chan struct{}),
		Ctx:                 ctx,
		cancelCtx:           cancelCtx,
		stopFlag:            abool.NewBool(false),
		ctrlFuncRunning:     abool.NewBool(false),
		workerCnt:           &workerCnt,
		taskCnt:             &taskCnt,
		microTaskCnt:        &microTaskCnt,
		eventHooks:          make(map[string]*eventHooks),
		depNames:            dependencies,
	}

	newModule.Sleep(false)

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

// EnterSleepMode enables or disables sleep mode for all the modules.
func EnterSleepMode(enabled bool) {
	// Check if differs with the old state.
	set := sleepMode.SetToIf(!enabled, enabled)
	if !set {
		return
	}

	// Update all modules
	for _, m := range modules {
		m.Sleep(enabled)
	}

	// Send signal to the task schedular.
	if !enabled {
		taskSchedulerSleepModeExitChannel <- struct{}{}
	}
}
