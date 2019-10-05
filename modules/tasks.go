package modules

import (
	"container/list"
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tevino/abool"

	"github.com/safing/portbase/log"
)

// Task is managed task bound to a module.
type Task struct {
	name   string
	module *Module
	taskFn func(context.Context, *Task)

	queued     bool
	canceled   bool
	executing  bool
	cancelFunc func()

	executeAt time.Time
	repeat    time.Duration
	maxDelay  time.Duration

	queueElement            *list.Element
	prioritizedQueueElement *list.Element
	scheduleListElement     *list.Element

	lock sync.Mutex
}

var (
	taskQueue            = list.New()
	prioritizedTaskQueue = list.New()
	queuesLock           sync.Mutex
	queueWg              sync.WaitGroup

	taskSchedule = list.New()
	scheduleLock sync.Mutex

	waitForever chan time.Time

	queueIsFilled                = make(chan struct{}, 1) // kick off queue handler
	recalculateNextScheduledTask = make(chan struct{}, 1)
	taskTimeslot                 = make(chan struct{})
)

const (
	maxTimeslotWait   = 30 * time.Second
	minRepeatDuration = 1 * time.Minute
	maxExecutionWait  = 1 * time.Minute
	defaultMaxDelay   = 1 * time.Minute
)

// NewTask creates a new task with a descriptive name (non-unique), a optional deadline, and the task function to be executed. You must call one of Queue, Prioritize, StartASAP, Schedule or Repeat in order to have the Task executed.
func (m *Module) NewTask(name string, fn func(context.Context, *Task)) *Task {
	if m == nil {
		log.Errorf(`modules: cannot create task "%s" with nil module`, name)
	}

	return &Task{
		name:     name,
		module:   m,
		taskFn:   fn,
		maxDelay: defaultMaxDelay,
	}
}

func (t *Task) isActive() bool {
	if t.module == nil {
		return false
	}

	return !t.canceled && !t.module.ShutdownInProgress()
}

func (t *Task) prepForQueueing() (ok bool) {
	if !t.isActive() {
		return false
	}

	t.queued = true
	if t.maxDelay != 0 {
		t.executeAt = time.Now().Add(t.maxDelay)
		t.addToSchedule()
	}

	return true
}

func notifyQueue() {
	select {
	case queueIsFilled <- struct{}{}:
	default:
	}
}

// Queue queues the Task for execution.
func (t *Task) Queue() *Task {
	t.lock.Lock()
	if !t.prepForQueueing() {
		t.lock.Unlock()
		return t
	}
	t.lock.Unlock()

	if t.queueElement == nil {
		queuesLock.Lock()
		t.queueElement = taskQueue.PushBack(t)
		queuesLock.Unlock()
	}

	notifyQueue()
	return t
}

// Prioritize puts the task in the prioritized queue.
func (t *Task) Prioritize() *Task {
	t.lock.Lock()
	if !t.prepForQueueing() {
		t.lock.Unlock()
		return t
	}
	t.lock.Unlock()

	if t.prioritizedQueueElement == nil {
		queuesLock.Lock()
		t.prioritizedQueueElement = prioritizedTaskQueue.PushBack(t)
		queuesLock.Unlock()
	}

	notifyQueue()
	return t
}

// StartASAP schedules the task to be executed next.
func (t *Task) StartASAP() *Task {
	t.lock.Lock()
	if !t.prepForQueueing() {
		t.lock.Unlock()
		return t
	}
	t.lock.Unlock()

	queuesLock.Lock()
	if t.prioritizedQueueElement == nil {
		t.prioritizedQueueElement = prioritizedTaskQueue.PushFront(t)
	} else {
		prioritizedTaskQueue.MoveToFront(t.prioritizedQueueElement)
	}
	queuesLock.Unlock()

	notifyQueue()
	return t
}

// MaxDelay sets a maximum delay within the task should be executed from being queued. Scheduled tasks are queued when they are triggered. The default delay is 3 minutes.
func (t *Task) MaxDelay(maxDelay time.Duration) *Task {
	t.lock.Lock()
	t.maxDelay = maxDelay
	t.lock.Unlock()
	return t
}

// Schedule schedules the task for execution at the given time.
func (t *Task) Schedule(executeAt time.Time) *Task {
	t.lock.Lock()
	t.executeAt = executeAt
	t.addToSchedule()
	t.lock.Unlock()
	return t
}

// Repeat sets the task to be executed in endless repeat at the specified interval. First execution will be after interval. Minimum repeat interval is one minute.
func (t *Task) Repeat(interval time.Duration) *Task {
	// check minimum interval duration
	if interval < minRepeatDuration {
		interval = minRepeatDuration
	}

	t.lock.Lock()
	t.repeat = interval
	t.executeAt = time.Now().Add(t.repeat)
	t.addToSchedule()
	t.lock.Unlock()

	return t
}

// Cancel cancels the current and any future execution of the Task. This is not reversible by any other functions.
func (t *Task) Cancel() {
	t.lock.Lock()
	t.canceled = true
	if t.cancelFunc != nil {
		t.cancelFunc()
	}
	t.lock.Unlock()
}

func (t *Task) runWithLocking() {
	if t.module == nil {
		return
	}

	// wait for good timeslot regarding microtasks
	select {
	case <-taskTimeslot:
	case <-time.After(maxTimeslotWait):
	}

	t.lock.Lock()

	// check state, return if already executing or inactive
	if t.executing || !t.isActive() {
		t.lock.Unlock()
		return
	}
	t.executing = true

	// get list elements
	queueElement := t.queueElement
	prioritizedQueueElement := t.prioritizedQueueElement
	scheduleListElement := t.scheduleListElement

	// create context
	var taskCtx context.Context
	taskCtx, t.cancelFunc = context.WithCancel(t.module.Ctx)

	t.lock.Unlock()

	// remove from lists
	if queueElement != nil {
		queuesLock.Lock()
		taskQueue.Remove(t.queueElement)
		queuesLock.Unlock()
		t.lock.Lock()
		t.queueElement = nil
		t.lock.Unlock()
	}
	if prioritizedQueueElement != nil {
		queuesLock.Lock()
		prioritizedTaskQueue.Remove(t.prioritizedQueueElement)
		queuesLock.Unlock()
		t.lock.Lock()
		t.prioritizedQueueElement = nil
		t.lock.Unlock()
	}
	if scheduleListElement != nil {
		scheduleLock.Lock()
		taskSchedule.Remove(t.scheduleListElement)
		scheduleLock.Unlock()
		t.lock.Lock()
		t.scheduleListElement = nil
		t.lock.Unlock()
	}

	// add to queue workgroup
	queueWg.Add(1)

	go t.executeWithLocking(taskCtx, t.cancelFunc)
	go func() {
		select {
		case <-taskCtx.Done():
		case <-time.After(maxExecutionWait):
		}
		// complete queue worker (early) to allow next worker
		queueWg.Done()
	}()
}

func (t *Task) executeWithLocking(ctx context.Context, cancelFunc func()) {
	// start for module
	// hint: only queueWg global var is important for scheduling, others can be set here
	atomic.AddInt32(t.module.taskCnt, 1)
	t.module.waitGroup.Add(1)

	defer func() {
		// recover from panic
		panicVal := recover()
		if panicVal != nil {
			me := t.module.NewPanicError(t.name, "task", panicVal)
			me.Report()
			log.Errorf("%s: task %s panicked: %s\n%s", t.module.Name, t.name, panicVal, me.StackTrace)
		}

		// finish for module
		atomic.AddInt32(t.module.taskCnt, -1)
		t.module.waitGroup.Done()

		// reset
		t.lock.Lock()
		// reset state
		t.executing = false
		t.queued = false
		// repeat?
		if t.isActive() && t.repeat != 0 {
			t.executeAt = time.Now().Add(t.repeat)
			t.addToSchedule()
		}
		t.lock.Unlock()

		// notify that we finished
		cancelFunc()
	}()

	// run
	t.taskFn(ctx, t)
}

func (t *Task) getExecuteAtWithLocking() time.Time {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.executeAt
}

func (t *Task) addToSchedule() {
	scheduleLock.Lock()
	defer scheduleLock.Unlock()
	// defer printTaskList(taskSchedule) // for debugging

	// notify scheduler
	defer func() {
		select {
		case recalculateNextScheduledTask <- struct{}{}:
		default:
		}
	}()

	// insert task into schedule
	for e := taskSchedule.Front(); e != nil; e = e.Next() {
		// check for self
		eVal := e.Value.(*Task)
		if eVal == t {
			continue
		}
		// compare
		if t.executeAt.Before(eVal.getExecuteAtWithLocking()) {
			// insert/move task
			if t.scheduleListElement == nil {
				t.scheduleListElement = taskSchedule.InsertBefore(t, e)
			} else {
				taskSchedule.MoveBefore(t.scheduleListElement, e)
			}
			return
		}
	}

	// add/move to end
	if t.scheduleListElement == nil {
		t.scheduleListElement = taskSchedule.PushBack(t)
	} else {
		taskSchedule.MoveToBack(t.scheduleListElement)
	}
}

func waitUntilNextScheduledTask() <-chan time.Time {
	scheduleLock.Lock()
	defer scheduleLock.Unlock()

	if taskSchedule.Len() > 0 {
		return time.After(time.Until(taskSchedule.Front().Value.(*Task).executeAt))
	}
	return waitForever
}

var (
	taskQueueHandlerStarted    = abool.NewBool(false)
	taskScheduleHandlerStarted = abool.NewBool(false)
)

func taskQueueHandler() {
	// only ever start once
	if !taskQueueHandlerStarted.SetToIf(false, true) {
		return
	}

	for {
		// wait
		select {
		case <-shutdownSignal:
			return
		case <-queueIsFilled:
		}

		// execute
	execLoop:
		for {
			// wait for execution slot
			queueWg.Wait()

			// check for shutdown
			if shutdownSignalClosed.IsSet() {
				return
			}

			// get next Task
			queuesLock.Lock()
			e := prioritizedTaskQueue.Front()
			if e != nil {
				prioritizedTaskQueue.Remove(e)
			} else {
				e = taskQueue.Front()
				if e != nil {
					taskQueue.Remove(e)
				}
			}
			queuesLock.Unlock()

			// lists are empty
			if e == nil {
				break execLoop
			}

			// value -> Task
			t := e.Value.(*Task)
			// run
			t.runWithLocking()
		}
	}
}

func taskScheduleHandler() {
	// only ever start once
	if !taskScheduleHandlerStarted.SetToIf(false, true) {
		return
	}

	for {
		select {
		case <-shutdownSignal:
			return
		case <-recalculateNextScheduledTask:
		case <-waitUntilNextScheduledTask():
			// get first task in schedule
			scheduleLock.Lock()
			e := taskSchedule.Front()
			scheduleLock.Unlock()
			t := e.Value.(*Task)

			// process Task
			if t.queued {
				// already queued and maxDelay reached
				t.runWithLocking()
			} else {
				// place in front of prioritized queue
				t.StartASAP()
			}
		}
	}
}

func printTaskList(*list.List) { //nolint:unused,deadcode // for debugging, NOT production use
	fmt.Println("Modules Task List:")
	for e := taskSchedule.Front(); e != nil; e = e.Next() {
		t, ok := e.Value.(*Task)
		if ok {
			fmt.Printf(
				"%s:%s qu=%v ca=%v exec=%v at=%s rep=%s delay=%s\n",
				t.module.Name,
				t.name,
				t.queued,
				t.canceled,
				t.executing,
				t.executeAt,
				t.repeat,
				t.maxDelay,
			)
		}
	}
}
