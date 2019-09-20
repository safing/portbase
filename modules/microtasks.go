package modules

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/safing/portbase/log"
	"github.com/tevino/abool"
)

// TODO: getting some errors when in nanosecond precision for tests:
// (1) panic: sync: WaitGroup is reused before previous Wait has returned - should theoretically not happen
// (2) sometimes there seems to some kind of race condition stuff, the test hangs and does not complete

var (
	microTasks           *int32
	microTasksThreshhold *int32
	microTaskFinished    = make(chan struct{}, 1)

	mediumPriorityClearance = make(chan struct{})
	lowPriorityClearance    = make(chan struct{})

	triggerLogWriting = log.TriggerWriterChannel()
)

const (
	mediumPriorityMaxDelay = 1 * time.Second
	lowPriorityMaxDelay    = 3 * time.Second
)

func init() {
	var microTasksVal int32
	microTasks = &microTasksVal
	var microTasksThreshholdVal int32
	microTasksThreshhold = &microTasksThreshholdVal
}

// SetMaxConcurrentMicroTasks sets the maximum number of microtasks that should be run concurrently.
func SetMaxConcurrentMicroTasks(n int) {
	if n < 4 {
		atomic.StoreInt32(microTasksThreshhold, 4)
	} else {
		atomic.StoreInt32(microTasksThreshhold, int32(n))
	}
}

// StartMicroTask starts a new MicroTask with high priority. It will start immediately. The given function will be executed and panics caught. The supplied name should be a constant - the variable should never change as it won't be copied.
func (m *Module) StartMicroTask(name *string, fn func(context.Context) error) error {
	atomic.AddInt32(microTasks, 1)
	return m.runMicroTask(name, fn)
}

// StartMediumPriorityMicroTask starts a new MicroTask with medium priority. It will wait until given a go (max 3 seconds). The given function will be executed and panics caught. The supplied name should be a constant - the variable should never change as it won't be copied.
func (m *Module) StartMediumPriorityMicroTask(name *string, fn func(context.Context) error) error {
	// check if we can go immediately
	select {
	case <-mediumPriorityClearance:
	default:
		// wait for go or max delay
		select {
		case <-mediumPriorityClearance:
		case <-time.After(mediumPriorityMaxDelay):
		}
	}
	return m.runMicroTask(name, fn)
}

// StartLowPriorityMicroTask starts a new MicroTask with low priority. It will wait until given a go (max 15 seconds). The given function will be executed and panics caught. The supplied name should be a constant - the variable should never change as it won't be copied.
func (m *Module) StartLowPriorityMicroTask(name *string, fn func(context.Context) error) error {
	// check if we can go immediately
	select {
	case <-lowPriorityClearance:
	default:
		// wait for go or max delay
		select {
		case <-lowPriorityClearance:
		case <-time.After(lowPriorityMaxDelay):
		}
	}
	return m.runMicroTask(name, fn)
}

func (m *Module) runMicroTask(name *string, fn func(context.Context) error) (err error) {
	// start for module
	// hint: only microTasks global var is important for scheduling, others can be set here
	atomic.AddInt32(m.microTaskCnt, 1)
	m.waitGroup.Add(1)

	// set up recovery
	defer func() {
		// recover from panic
		panicVal := recover()
		if panicVal != nil {
			me := m.NewPanicError(*name, "microtask", panicVal)
			me.Report()
			log.Errorf("%s: microtask %s panicked: %s\n%s", m.Name, *name, panicVal, me.StackTrace)
			err = me
		}

		// finish for module
		atomic.AddInt32(m.microTaskCnt, -1)
		m.waitGroup.Done()

		// finish and possibly trigger next task
		atomic.AddInt32(microTasks, -1)
		select {
		case microTaskFinished <- struct{}{}:
		default:
		}
	}()

	// run
	err = fn(m.Ctx)
	return //nolint:nakedret // need to use named return val in order to change in defer
}

var (
	microTaskSchedulerStarted = abool.NewBool(false)
)

func microTaskScheduler() {
	// only ever start once
	if !microTaskSchedulerStarted.SetToIf(false, true) {
		return
	}

microTaskManageLoop:
	for {
		if shutdownSignalClosed.IsSet() {
			close(mediumPriorityClearance)
			close(lowPriorityClearance)
			return
		}

		if atomic.LoadInt32(microTasks) < atomic.LoadInt32(microTasksThreshhold) { // space left for firing task
			select {
			case mediumPriorityClearance <- struct{}{}:
			default:
				select {
				case taskTimeslot <- struct{}{}:
					continue microTaskManageLoop
				case triggerLogWriting <- struct{}{}:
					continue microTaskManageLoop
				case mediumPriorityClearance <- struct{}{}:
				case lowPriorityClearance <- struct{}{}:
				}
			}
			// increase task counter
			atomic.AddInt32(microTasks, 1)
		} else {
			// wait for signal that a task was completed
			<-microTaskFinished
		}

	}
}
