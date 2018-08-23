// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package taskmanager

import (
	"container/list"
	"time"

	"github.com/tevino/abool"
)

type Task struct {
	name     string
	start    chan bool
	started  *abool.AtomicBool
	schedule *time.Time
}

var taskQueue *list.List
var prioritizedTaskQueue *list.List
var addToQueue chan *Task
var addToPrioritizedQueue chan *Task
var addAsNextTask chan *Task

var finishedQueuedTask chan bool
var queuedTaskRunning *abool.AtomicBool

var getQueueLengthREQ chan bool
var getQueueLengthREP chan int

func newUnqeuedTask(name string) *Task {
	t := &Task{
		name,
		make(chan bool),
		abool.NewBool(false),
		nil,
	}
	return t
}

func NewQueuedTask(name string) *Task {
	t := newUnqeuedTask(name)
	addToQueue <- t
	return t
}

func NewPrioritizedQueuedTask(name string) *Task {
	t := newUnqeuedTask(name)
	addToPrioritizedQueue <- t
	return t
}

func (t *Task) addToPrioritizedQueue() {
	addToPrioritizedQueue <- t
}

func (t *Task) WaitForStart() chan bool {
	return t.start
}

func (t *Task) StartAnyway() {
	addAsNextTask <- t
}

func (t *Task) Done() {
	if !t.started.SetToIf(false, true) {
		finishedQueuedTask <- true
	}
}

func TotalQueuedTasks() int {
	getQueueLengthREQ <- true
	return <-getQueueLengthREP
}

func checkQueueStatus() {
	if queuedTaskRunning.SetToIf(false, true) {
		finishedQueuedTask <- true
	}
}

func fireNextTask() {

	if prioritizedTaskQueue.Len() > 0 {
		for e := prioritizedTaskQueue.Front(); prioritizedTaskQueue.Len() > 0; e.Next() {
			t := e.Value.(*Task)
			prioritizedTaskQueue.Remove(e)
			if t.started.SetToIf(false, true) {
				close(t.start)
				return
			}
		}
	}

	if taskQueue.Len() > 0 {
		for e := taskQueue.Front(); taskQueue.Len() > 0; e.Next() {
			t := e.Value.(*Task)
			taskQueue.Remove(e)
			if t.started.SetToIf(false, true) {
				close(t.start)
				return
			}
		}
	}

	queuedTaskRunning.UnSet()

}

func init() {

	taskQueue = list.New()
	prioritizedTaskQueue = list.New()
	addToQueue = make(chan *Task, 1)
	addToPrioritizedQueue = make(chan *Task, 1)
	addAsNextTask = make(chan *Task, 1)

	finishedQueuedTask = make(chan bool, 1)
	queuedTaskRunning = abool.NewBool(false)

	getQueueLengthREQ = make(chan bool, 1)
	getQueueLengthREP = make(chan int, 1)

	go func() {

		for {
			select {
			case <-shutdownSignal:
				// TODO: work off queue?
				return
			case <-getQueueLengthREQ:
				// TODO: maybe clean queues before replying
				if queuedTaskRunning.IsSet() {
					getQueueLengthREP <- prioritizedTaskQueue.Len() + taskQueue.Len() + 1
				} else {
					getQueueLengthREP <- prioritizedTaskQueue.Len() + taskQueue.Len()
				}
			case t := <-addToQueue:
				taskQueue.PushBack(t)
				checkQueueStatus()
			case t := <-addToPrioritizedQueue:
				prioritizedTaskQueue.PushBack(t)
				checkQueueStatus()
			case t := <-addAsNextTask:
				prioritizedTaskQueue.PushFront(t)
				checkQueueStatus()
			case <-finishedQueuedTask:
				fireNextTask()
			}
		}

	}()

}
