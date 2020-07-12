package modules

import (
	"context"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/safing/portbase/log"
)

// TODO: getting some errors when in nanosecond precision for tests:
// (1) panic: sync: WaitGroup is reused before previous Wait has returned - should theoretically not happen
// (2) sometimes there seems to some kind of race condition stuff, the test hangs and does not complete

var emptyStructVal = reflect.ValueOf(struct{}{})

// MicroTaskFunc defines the function prototype for micro tasks.
type MicroTaskFunc func(ctx context.Context) error

// MicroTaskScheduler schedules execution of mirco tasks. In addition
// to normal micro tasks, the scheduler supports medium and low priority
// tasks that are only executed if execution slots are available. While
// all normal (high priority) tasks are executed immediately low and medium
// priority tasks are only run if less tasks than the configured concurrency
// limit are currently active. The execution of each tasks is wrapped in
// with recovery and reporting methods.
type MicroTaskScheduler struct {
	// activeTasks counts the number of currently running
	// tasks.
	// Note that high priority tasks do not honor the task
	// scheduler and are executed immediately.
	activeTasks int32

	// concurrencyLimit limits the number of tasks that are
	// allowed to be executed concurrently.
	concurrencyLimit int32

	// taskFinishNotifier is used to signal the scheduler
	// goroutine that another task may run. Senders must not
	// expect the channel to be read and use a select block
	// with a default.
	taskFinishNotifier chan struct{}

	// mediumPriorityClearance is used to trigger a medium priority
	// task waiting for execution.
	mediumPriorityClearance chan struct{}

	// lowPriorityClearance is used to trigger a low priority
	// task waiting for execution.
	lowPriorityClearance chan struct{}

	// idleNotifiersLock is used to protect idleNotifiers
	idleNotifiersLock sync.Mutex

	// idleNotifiers is a slice of channels that are
	// notified when the task scheduler is idle.
	idleNotifiers []chan<- struct{}

	// wg can be used to wait for all currently running tasks
	// to finish.
	wg sync.WaitGroup

	// start is used to start the micro-task scheduler once.
	start sync.Once
}

// Wait waits until the micro task scheduler has shutdown.
func (ts *MicroTaskScheduler) Wait() {
	ts.wg.Wait()
}

// NewMicroTaskScheduler returns a new microtask scheduler.
func NewMicroTaskScheduler() *MicroTaskScheduler {
	return &MicroTaskScheduler{
		taskFinishNotifier:      make(chan struct{}),
		mediumPriorityClearance: make(chan struct{}),
		lowPriorityClearance:    make(chan struct{}),
	}
}

// Start starts the microtask scheduler. If the scheduler has already
// been started Start() is a no-op. If ctx is cancelled the task
// schedule will shutdown and cannot be started again.
func (ts *MicroTaskScheduler) Start(ctx context.Context) {
	ts.start.Do(func() {
		ts.wg.Add(1)
		go ts.scheduler(ctx)
	})
}

// SetMaxConcurrentMicroTasks sets the maximum number of tasks
// that are allowed to execute concurrently.
func (ts *MicroTaskScheduler) SetMaxConcurrentMicroTasks(limit int) {
	if limit < 4 {
		limit = 4
	}
	atomic.StoreInt32(&ts.concurrencyLimit, int32(limit))
}

// RunNow executes fn as a micro task right now without waiting for a
// dedicated execution slot.
func (ts *MicroTaskScheduler) RunNow(m *Module, name string, fn MicroTaskFunc) error {
	now := make(chan struct{})
	close(now)
	atomic.AddInt32(&ts.activeTasks, 1) // we start immediately so the scheduler doesn't know
	return ts.waitAndRun(m, now, time.Second, name, fn)
}

// RunLowPriority waits for a low-priority execution slot and executes fn.
func (ts *MicroTaskScheduler) RunLowPriority(m *Module, name string, fn MicroTaskFunc) error {
	return ts.waitAndRun(m, ts.lowPriorityClearance, lowPriorityMaxDelay, name, fn)
}

// RunMediumPriority waits for a medium-priority execution slot and
// executes fn.
func (ts *MicroTaskScheduler) RunMediumPriority(m *Module, name string, fn MicroTaskFunc) error {
	return ts.waitAndRun(m, ts.mediumPriorityClearance, mediumPriorityMaxDelay, name, fn)
}

// AddIdleNotifier adds ch as an idle notifier. Whenever the micro task scheduler becomes
// idle it tries to notify one of the listening channels. The scheduler will block until
// a notifier becomes active (receives on ch). Most of the time callers should use
// unbuffered channels for notifiers.
//
// Example:
//
//		idle := make(chan struct{})
//		go func() {
//			for _ = range idle {
//				// Microtask scheduler is idle ...
//			}
//		}()
//		ts.AddIdleNotifier(idle)
//
func (ts *MicroTaskScheduler) AddIdleNotifier(ch chan<- struct{}) {
	ts.idleNotifiersLock.Lock()
	defer ts.idleNotifiersLock.Unlock()

	ts.idleNotifiers = append(ts.idleNotifiers, ch)
}

// runMicroTask executes immediately executes fn with task recovery and
// scheduling notifiers. The caller is responsible of increasing ts.wg
// before executing runMicroTask.
func (ts *MicroTaskScheduler) runMicroTask(m *Module, name string, fn MicroTaskFunc) (err error) {
	defer Recoverf(m, &err, name, "microtask")
	defer ts.wg.Done()
	defer func() {
		atomic.AddInt32(&ts.activeTasks, -1)
		select {
		case ts.taskFinishNotifier <- struct{}{}:
		default:
		}
	}()

	return fn(m.Ctx)
}

// waitAndRun increases the wait group of ts and waits for either trigger or maxDelay to allow execution
// of fn. Once triggered, waitAndRun calls runMicroTask.
func (ts *MicroTaskScheduler) waitAndRun(m *Module, trigger <-chan struct{}, maxDelay time.Duration, name string, fn MicroTaskFunc) (err error) {
	ts.wg.Add(1)

	select {
	case <-trigger:
	default:
		select {
		case <-time.After(maxDelay):
		case <-trigger:
		case <-m.Ctx.Done():
			// NOTE(ppacher): legacy task scheduler (<= v0.7.2) did not
			// wait for context cancellation and risked racing
			// between starting a task and blocking on the wait group.
			// It's not sure if we can count that as backwards
			// compatible.
			ts.wg.Done()
			return ErrShuttingDown
		}

		if m.Ctx.Err() != nil {
			// <-trigger may fire if it's closed during scheduler shutdown.
			return ErrShuttingDown
		}
	}

	return ts.runMicroTask(m, name, fn)
}

func (ts *MicroTaskScheduler) canFire() bool {
	return atomic.LoadInt32(&ts.activeTasks) < atomic.LoadInt32(&ts.concurrencyLimit)
}

func (ts *MicroTaskScheduler) notifyAnyWaiter(ctx context.Context) (isTask bool) {
	ts.idleNotifiersLock.Lock()
	cases := make([]reflect.SelectCase, len(ts.idleNotifiers)+3)

	cases[0] = reflect.SelectCase{
		Dir:  reflect.SelectSend,
		Chan: reflect.ValueOf(ts.mediumPriorityClearance),
		Send: emptyStructVal,
	}
	cases[1] = reflect.SelectCase{
		Dir:  reflect.SelectSend,
		Chan: reflect.ValueOf(ts.lowPriorityClearance),
		Send: emptyStructVal,
	}
	cases[2] = reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(ctx.Done()),
	}

	for idx, ch := range ts.idleNotifiers {
		cases[idx+3] = reflect.SelectCase{
			Dir:  reflect.SelectSend,
			Chan: reflect.ValueOf(ch),
			Send: emptyStructVal,
		}
	}
	ts.idleNotifiersLock.Unlock()

	chosen, _, _ := reflect.Select(cases)
	switch chosen {
	case 0, 1: // medium or low priority clearance
		return true
	case 2: // context cancelled
		return false
	default: // external idle notifiers
		return false
	}
}

func (ts *MicroTaskScheduler) scheduler(ctx context.Context) {
	defer ts.wg.Done()
	defer close(ts.lowPriorityClearance)
	defer close(ts.mediumPriorityClearance)

	for {
		if ts.canFire() {
			select {
			case <-ctx.Done():
				return

			case ts.mediumPriorityClearance <- struct{}{}:
				atomic.AddInt32(&ts.activeTasks, 1)

			default:
				// trigger the first one to wait for a slot
				if isTask := ts.notifyAnyWaiter(ctx); isTask {
					// increase task counter
					atomic.AddInt32(&ts.activeTasks, 1)
				}
			}
		} else {
			// wait for signal that a task was completed
			select {
			case <-ctx.Done():
				return
			case <-ts.taskFinishNotifier:
			case <-time.After(1 * time.Second):
			}
		}
	}
}

// DefaultMicroTaskScheduler is the default micro task scheduler.
var DefaultMicroTaskScheduler *MicroTaskScheduler

const (
	mediumPriorityMaxDelay = 1 * time.Second
	lowPriorityMaxDelay    = 3 * time.Second
)

func init() {
	DefaultMicroTaskScheduler = NewMicroTaskScheduler()
	// trigger log writting when the microtask scheduler
	// is idle.
	DefaultMicroTaskScheduler.AddIdleNotifier(log.TriggerWriterChannel())

	// allow normal tasks to execute if the micro-task scheduler is
	// idle.
	DefaultMicroTaskScheduler.AddIdleNotifier(taskTimeslot)
}

// SetMaxConcurrentMicroTasks sets the maximum number of microtasks that should be run
// concurrently. It uses the default micro task scheduler instance.
func SetMaxConcurrentMicroTasks(n int) {
	DefaultMicroTaskScheduler.SetMaxConcurrentMicroTasks(n)
}

// StartMicroTask starts a new MicroTask with high priority. It will start immediately.
// The call starts a new goroutine and returns immediately. The given function will
// be executed and panics caught. The supplied name must not be changed.
func (m *Module) StartMicroTask(name *string, fn func(context.Context) error) {
	go func() {
		if err := m.RunMicroTask(name, fn); err != nil {
			log.Warningf("%s: microtask %s failed: %s", m.Name, name, err)
		}
	}()
}

// StartMediumPriorityMicroTask starts a new MicroTask with medium priority.
// The call starts a new goroutine and returns immediately. It will wait until
// a slot becomes available (max 3 seconds). The given function will be
// executed and panics caught. The supplied name must not be changed.
func (m *Module) StartMediumPriorityMicroTask(name *string, fn func(context.Context) error) {
	go func() {
		if err := m.RunMediumPriorityMicroTask(name, fn); err != nil {
			log.Warningf("%s: microtask %s failed: %s", m.Name, name, err)
		}
	}()
}

// StartLowPriorityMicroTask starts a new MicroTask with low priority.
// The call starts a new goroutine and returns immediately. It will wait
// until a slot becomes available (max 15 seconds). The given function
// will be executed and panics caught. The supplied name must not be changed.
func (m *Module) StartLowPriorityMicroTask(name *string, fn func(context.Context) error) {
	go func() {
		if err := m.RunLowPriorityMicroTask(name, fn); err != nil {
			log.Warningf("%s: microtask %s failed: %s", m.Name, name, err)
		}
	}()
}

// RunMicroTask runs a new MicroTask with high priority.
// It will start immediately. The call blocks until finished.
// The given function will be executed and panics caught.
// The supplied name must not be changed.
func (m *Module) RunMicroTask(name *string, fn func(context.Context) error) error {
	defer m.countTask()()
	return DefaultMicroTaskScheduler.RunNow(m, *name, fn)
}

// RunMediumPriorityMicroTask runs a new MicroTask with medium priority.
// It will wait until a slot becomes available (max 3 seconds). The call blocks
// until finished. The given function will be executed and panics caught.
// The supplied name must not be changed.
func (m *Module) RunMediumPriorityMicroTask(name *string, fn func(context.Context) error) error {
	defer m.countTask()()
	return DefaultMicroTaskScheduler.RunMediumPriority(m, *name, fn)
}

// RunLowPriorityMicroTask runs a new MicroTask with low priority.
// It will wait until a slot becomes available (max 15 seconds).
// The call blocks until finished. The given function will be executed
// and panics caught. The supplied name must not be changed.
func (m *Module) RunLowPriorityMicroTask(name *string, fn func(context.Context) error) error {
	defer m.countTask()()
	return DefaultMicroTaskScheduler.RunLowPriority(m, *name, fn)
}

func (m *Module) countTask() func() {
	atomic.AddInt32(m.microTaskCnt, 1)
	m.waitGroup.Add(1)
	return func() {
		// finish for module
		atomic.AddInt32(m.microTaskCnt, -1)
		m.waitGroup.Done()
	}
}
