package modules

import (
	"context"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/tevino/abool"

	"github.com/safing/portbase/log"
)

// TODO: getting some errors when in nanosecond precision for tests:
// (1) panic: sync: WaitGroup is reused before previous Wait has returned - should theoretically not happen
// (2) sometimes there seems to some kind of race condition stuff, the test hangs and does not complete
// NOTE: These might be resolved by the switch to clearance queues.

var (
	microTasks           *int32
	microTasksThreshhold *int32
	microTaskFinished    = make(chan struct{}, 1)
)

const (
	defaultMediumPriorityMaxDelay = 1 * time.Second
	defaultLowPriorityMaxDelay    = 3 * time.Second
)

func init() {
	var microTasksVal int32
	microTasks = &microTasksVal

	microTasksThreshholdVal := int32(runtime.GOMAXPROCS(0) * 2)
	microTasksThreshhold = &microTasksThreshholdVal
}

// SetMaxConcurrentMicroTasks sets the maximum number of microtasks that should
// be run concurrently. The modules system initializes it with GOMAXPROCS.
// The minimum is 2.
func SetMaxConcurrentMicroTasks(n int) {
	if n < 2 {
		atomic.StoreInt32(microTasksThreshhold, 2)
	} else {
		atomic.StoreInt32(microTasksThreshhold, int32(n))
	}
}

// StartHighPriorityMicroTask starts a new MicroTask with high priority.
// It will start immediately.
// The call starts a new goroutine and returns immediately.
// The given function will be executed and panics caught.
func (m *Module) StartHighPriorityMicroTask(name string, fn func(context.Context) error) {
	go func() {
		err := m.RunHighPriorityMicroTask(name, fn)
		if err != nil {
			log.Warningf("%s: microtask %s failed: %s", m.Name, name, err)
		}
	}()
}

// StartMicroTask starts a new MicroTask with medium priority.
// The call starts a new goroutine and returns immediately.
// It will wait until a slot becomes available while respecting maxDelay.
// You can also set maxDelay to 0 to use the default value of 1 second.
// The given function will be executed and panics caught.
func (m *Module) StartMicroTask(name string, maxDelay time.Duration, fn func(context.Context) error) {
	go func() {
		err := m.RunMicroTask(name, maxDelay, fn)
		if err != nil {
			log.Warningf("%s: microtask %s failed: %s", m.Name, name, err)
		}
	}()
}

// StartLowPriorityMicroTask starts a new MicroTask with low priority.
// The call starts a new goroutine and returns immediately.
// It will wait until a slot becomes available while respecting maxDelay.
// You can also set maxDelay to 0 to use the default value of 3 seconds.
// The given function will be executed and panics caught.
func (m *Module) StartLowPriorityMicroTask(name string, maxDelay time.Duration, fn func(context.Context) error) {
	go func() {
		err := m.RunLowPriorityMicroTask(name, maxDelay, fn)
		if err != nil {
			log.Warningf("%s: microtask %s failed: %s", m.Name, name, err)
		}
	}()
}

// RunHighPriorityMicroTask starts a new MicroTask with high priority.
// The given function will be executed and panics caught.
// The call blocks until the given function finishes.
func (m *Module) RunHighPriorityMicroTask(name string, fn func(context.Context) error) error {
	if m == nil {
		log.Errorf(`modules: cannot start microtask "%s" with nil module`, name)
		return errNoModule
	}

	// Increase global counter here, as high priority tasks do not wait for clearance.
	atomic.AddInt32(microTasks, 1)
	return m.runMicroTask(name, fn)
}

// RunMicroTask starts a new MicroTask with medium priority.
// It will wait until a slot becomes available while respecting maxDelay.
// You can also set maxDelay to 0 to use the default value of 1 second.
// The given function will be executed and panics caught.
// The call blocks until the given function finishes.
func (m *Module) RunMicroTask(name string, maxDelay time.Duration, fn func(context.Context) error) error {
	if m == nil {
		log.Errorf(`modules: cannot start microtask "%s" with nil module`, name)
		return errNoModule
	}

	// Set default max delay, if not defined.
	if maxDelay <= 0 {
		maxDelay = defaultMediumPriorityMaxDelay
	}

	getMediumPriorityClearance(maxDelay)
	return m.runMicroTask(name, fn)
}

// RunLowPriorityMicroTask starts a new MicroTask with low priority.
// It will wait until a slot becomes available while respecting maxDelay.
// You can also set maxDelay to 0 to use the default value of 3 seconds.
// The given function will be executed and panics caught.
// The call blocks until the given function finishes.
func (m *Module) RunLowPriorityMicroTask(name string, maxDelay time.Duration, fn func(context.Context) error) error {
	if m == nil {
		log.Errorf(`modules: cannot start microtask "%s" with nil module`, name)
		return errNoModule
	}

	// Set default max delay, if not defined.
	if maxDelay <= 0 {
		maxDelay = defaultLowPriorityMaxDelay
	}

	getLowPriorityClearance(maxDelay)
	return m.runMicroTask(name, fn)
}

func (m *Module) runMicroTask(name string, fn func(context.Context) error) (err error) {
	// start for module
	// hint: only microTasks global var is important for scheduling, others can be set here
	atomic.AddInt32(m.microTaskCnt, 1)

	// set up recovery
	defer func() {
		// recover from panic
		panicVal := recover()
		if panicVal != nil {
			me := m.NewPanicError(name, "microtask", panicVal)
			me.Report()
			log.Errorf("%s: microtask %s panicked: %s", m.Name, name, panicVal)
			err = me
		}

		m.concludeMicroTask()
	}()

	// run
	err = fn(m.Ctx)
	return // Use named return val in order to change it in defer.
}

// SignalHighPriorityMicroTask signals the start of a new MicroTask with high priority.
// The returned "done" function SHOULD be called when the task has finished
// and MUST be called in any case. Failing to do so will have devastating effects.
// You can safely call "done" multiple times; additional calls do nothing.
func (m *Module) SignalHighPriorityMicroTask() (done func()) {
	if m == nil {
		log.Errorf("modules: cannot signal microtask with nil module")
		return
	}

	// Increase global counter here, as high priority tasks do not wait for clearance.
	atomic.AddInt32(microTasks, 1)
	return m.signalMicroTask()
}

// SignalMicroTask signals the start of a new MicroTask with medium priority.
// The call will wait until a slot becomes available while respecting maxDelay.
// You can also set maxDelay to 0 to use the default value of 1 second.
// The returned "done" function SHOULD be called when the task has finished
// and MUST be called in any case. Failing to do so will have devastating effects.
// You can safely call "done" multiple times; additional calls do nothing.
func (m *Module) SignalMicroTask(maxDelay time.Duration) (done func()) {
	if m == nil {
		log.Errorf("modules: cannot signal microtask with nil module")
		return
	}

	getMediumPriorityClearance(maxDelay)
	return m.signalMicroTask()
}

// SignalLowPriorityMicroTask signals the start of a new MicroTask with low priority.
// The call will wait until a slot becomes available while respecting maxDelay.
// You can also set maxDelay to 0 to use the default value of 1 second.
// The returned "done" function SHOULD be called when the task has finished
// and MUST be called in any case. Failing to do so will have devastating effects.
// You can safely call "done" multiple times; additional calls do nothing.
func (m *Module) SignalLowPriorityMicroTask(maxDelay time.Duration) (done func()) {
	if m == nil {
		log.Errorf("modules: cannot signal microtask with nil module")
		return
	}

	getLowPriorityClearance(maxDelay)
	return m.signalMicroTask()
}

func (m *Module) signalMicroTask() (done func()) {
	// Start microtask for module.
	// Global counter is set earlier as required for scheduling.
	atomic.AddInt32(m.microTaskCnt, 1)

	doneCalled := abool.New()
	return func() {
		if doneCalled.SetToIf(false, true) {
			m.concludeMicroTask()
		}
	}
}

func (m *Module) concludeMicroTask() {
	// Finish for module.
	atomic.AddInt32(m.microTaskCnt, -1)
	m.checkIfStopComplete()

	// Finish and possibly trigger next task.
	atomic.AddInt32(microTasks, -1)
	select {
	case microTaskFinished <- struct{}{}:
	default:
	}
}

var (
	clearanceQueueBaseSize  = 100
	clearanceQueueSize      = runtime.GOMAXPROCS(0) * clearanceQueueBaseSize
	mediumPriorityClearance = make(chan chan struct{}, clearanceQueueSize)
	lowPriorityClearance    = make(chan chan struct{}, clearanceQueueSize)

	triggerLogWriting = log.TriggerWriterChannel()

	microTaskSchedulerStarted = abool.NewBool(false)
)

func microTaskScheduler() {
	var clearanceSignal chan struct{}

	// Create ticker for max delay for checking clearances.
	recheck := time.NewTicker(1 * time.Second)
	defer recheck.Stop()

	// only ever start once
	if !microTaskSchedulerStarted.SetToIf(false, true) {
		return
	}

	// Debugging: Print current amount of microtasks.
	// go func() {
	// 	for {
	// 		time.Sleep(1 * time.Second)
	// 		log.Debugf("modules: microtasks: %d", atomic.LoadInt32(microTasks))
	// 	}
	// }()

	for {
		if shutdownFlag.IsSet() {
			go microTaskShutdownScheduler()
			return
		}

		// Check if there is space for one more microtask.
		if atomic.LoadInt32(microTasks) < atomic.LoadInt32(microTasksThreshhold) { // space left for firing task
			// Give Medium clearance.
			select {
			case clearanceSignal = <-mediumPriorityClearance:
			default:

				// Give Medium and Low clearance.
				select {
				case clearanceSignal = <-mediumPriorityClearance:
				case clearanceSignal = <-lowPriorityClearance:
				default:

					// Give Medium, Low and other clearancee.
					select {
					case clearanceSignal = <-mediumPriorityClearance:
					case clearanceSignal = <-lowPriorityClearance:
					case taskTimeslot <- struct{}{}:
					case triggerLogWriting <- struct{}{}:
					}
				}
			}

			// Send clearance signal and increase task counter.
			if clearanceSignal != nil {
				close(clearanceSignal)
				atomic.AddInt32(microTasks, 1)
			}
			clearanceSignal = nil
		} else {
			// wait for signal that a task was completed
			select {
			case <-microTaskFinished:
			case <-recheck.C:
			}
		}

	}
}

func microTaskShutdownScheduler() {
	var clearanceSignal chan struct{}

	for {
		// During shutdown, always give clearances immediately.
		select {
		case clearanceSignal = <-mediumPriorityClearance:
		case clearanceSignal = <-lowPriorityClearance:
		case taskTimeslot <- struct{}{}:
		case triggerLogWriting <- struct{}{}:
		}

		// Give clearance if requested.
		if clearanceSignal != nil {
			close(clearanceSignal)
			atomic.AddInt32(microTasks, 1)
		}
		clearanceSignal = nil
	}
}

func getMediumPriorityClearance(maxDelay time.Duration) {
	// Submit signal to scheduler.
	signal := make(chan struct{})
	select {
	case mediumPriorityClearance <- signal:
	default:
		select {
		case mediumPriorityClearance <- signal:
		case <-time.After(maxDelay):
			// Start without clearance and increase microtask counter.
			atomic.AddInt32(microTasks, 1)
			return
		}
	}
	// Wait for signal to start.
	select {
	case <-signal:
	default:
		select {
		case <-signal:
		case <-time.After(maxDelay):
			// Don't keep waiting for signal forever.
			// Don't increase microtask counter, as the signal was already submitted
			// and the counter will be increased by the scheduler.
		}
	}
}

func getLowPriorityClearance(maxDelay time.Duration) {
	// Submit signal to scheduler.
	signal := make(chan struct{})
	select {
	case lowPriorityClearance <- signal:
	default:
		select {
		case lowPriorityClearance <- signal:
		case <-time.After(maxDelay):
			// Start without clearance and increase microtask counter.
			atomic.AddInt32(microTasks, 1)
			return
		}
	}
	// Wait for signal to start.
	select {
	case <-signal:
	default:
		select {
		case <-signal:
		case <-time.After(maxDelay):
			// Don't keep waiting for signal forever.
			// Don't increase microtask counter, as the signal was already submitted
			// and the counter will be increased by the scheduler.
		}
	}
}
