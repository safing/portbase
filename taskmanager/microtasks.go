// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package taskmanager

import (
	"github.com/Safing/safing-core/modules"
	"sync/atomic"
	"time"

	"github.com/tevino/abool"
)

// TODO: getting some errors when in nanosecond precision for tests:
// (1) panic: sync: WaitGroup is reused before previous Wait has returned - should theoretically not happen
// (2) sometimes there seems to some kind of race condition stuff, the test hangs and does not complete

var microTasksModule *modules.Module
var closedChannel chan bool

var tasks *int32

var mediumPriorityClearance chan bool
var lowPriorityClearance chan bool
var veryLowPriorityClearance chan bool

var tasksDone chan bool
var tasksDoneFlag *abool.AtomicBool
var tasksWaiting chan bool
var tasksWaitingFlag *abool.AtomicBool

// StartMicroTask starts a new MicroTask. It will start immediately.
func StartMicroTask() {
	atomic.AddInt32(tasks, 1)
	tasksDoneFlag.UnSet()
}

// EndMicroTask MUST be always called when a MicroTask was previously started.
func EndMicroTask() {
	c := atomic.AddInt32(tasks, -1)
	if c < 1 {
		if tasksDoneFlag.SetToIf(false, true) {
			tasksDone <- true
		}
	}
}

func newTaskIsWaiting() {
	tasksWaiting <- true
}

// StartMediumPriorityMicroTask starts a new MicroTask (waiting its turn) if channel receives.
func StartMediumPriorityMicroTask() chan bool {
	if !microTasksModule.Active.IsSet() {
		return closedChannel
	}
	if tasksWaitingFlag.SetToIf(false, true) {
		defer newTaskIsWaiting()
	}
	return mediumPriorityClearance
}

// StartLowPriorityMicroTask starts a new MicroTask (waiting its turn) if channel receives.
func StartLowPriorityMicroTask() chan bool {
	if !microTasksModule.Active.IsSet() {
		return closedChannel
	}
	if tasksWaitingFlag.SetToIf(false, true) {
		defer newTaskIsWaiting()
	}
	return lowPriorityClearance
}

// StartVeryLowPriorityMicroTask starts a new MicroTask (waiting its turn) if channel receives.
func StartVeryLowPriorityMicroTask() chan bool {
	if !microTasksModule.Active.IsSet() {
		return closedChannel
	}
	if tasksWaitingFlag.SetToIf(false, true) {
		defer newTaskIsWaiting()
	}
	return veryLowPriorityClearance
}

func init() {

	microTasksModule = modules.Register("Taskmanager:MicroTasks", 3)
	closedChannel = make(chan bool, 0)
	close(closedChannel)

	var t int32 = 0
	tasks = &t

	mediumPriorityClearance = make(chan bool, 0)
	lowPriorityClearance = make(chan bool, 0)
	veryLowPriorityClearance = make(chan bool, 0)

	tasksDone = make(chan bool, 1)
	tasksDoneFlag = abool.NewBool(true)
	tasksWaiting = make(chan bool, 1)
	tasksWaitingFlag = abool.NewBool(false)

	timoutTimerDuration := 1 * time.Second
	// timoutTimer := time.NewTimer(timoutTimerDuration)

	go func() {
	microTaskManageLoop:
		for {

			// wait for an event to start new tasks
			if microTasksModule.Active.IsSet() {

				// reset timer
				// https://golang.org/pkg/time/#Timer.Reset
				// if !timoutTimer.Stop() {
				//   <-timoutTimer.C
				// }
				// timoutTimer.Reset(timoutTimerDuration)

				// wait for event to start a new task
				select {
				case <-tasksWaiting:
					if !tasksDoneFlag.IsSet() {
						continue microTaskManageLoop
					}
				case <-time.After(timoutTimerDuration):
				case <-tasksDone:
				case <-microTasksModule.Stop:
				}

			} else {

				// execute tasks until no tasks are waiting anymore
				if !tasksWaitingFlag.IsSet() {
					// wait until tasks are finished
					if !tasksDoneFlag.IsSet() {
						<-tasksDone
					}
					// signal module completion
					microTasksModule.StopComplete()
					// exit
					return
				}

			}

			// start new task, if none is started, check if we are shutting down
			select {
			case mediumPriorityClearance <- true:
				StartMicroTask()
			default:
				select {
				case lowPriorityClearance <- true:
					StartMicroTask()
				default:
					select {
					case veryLowPriorityClearance <- true:
						StartMicroTask()
					default:
						tasksWaitingFlag.UnSet()
					}
				}
			}

		}
	}()

}
