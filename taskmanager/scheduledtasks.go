package taskmanager

import (
	"container/list"
	"time"
)

var taskSchedule *list.List
var addToSchedule chan *Task
var waitForever chan time.Time

var getScheduleLengthREQ chan bool
var getScheduleLengthREP chan int

func NewScheduledTask(name string, schedule time.Time) *Task {
	t := newUnqeuedTask(name)
	t.schedule = &schedule
	addToSchedule <- t
	return t
}

func TotalScheduledTasks() int {
	getScheduleLengthREQ <- true
	return <-getScheduleLengthREP
}

func (t *Task) addToSchedule() {
	for e := taskSchedule.Back(); e != nil; e = e.Prev() {
		if t.schedule.After(*e.Value.(*Task).schedule) {
			taskSchedule.InsertAfter(t, e)
			return
		}
	}
	taskSchedule.PushFront(t)
}

func waitUntilNextScheduledTask() <-chan time.Time {
	if taskSchedule.Len() > 0 {
		return time.After(taskSchedule.Front().Value.(*Task).schedule.Sub(time.Now()))
	}
	return waitForever
}

func init() {

	taskSchedule = list.New()
	addToSchedule = make(chan *Task, 1)
	waitForever = make(chan time.Time, 1)

	getScheduleLengthREQ = make(chan bool, 1)
	getScheduleLengthREP = make(chan int, 1)

	go func() {

		for {
			select {
			case <-shutdownSignal:
				return
			case <-getScheduleLengthREQ:
				// TODO: maybe clean queues before replying
				getScheduleLengthREP <- prioritizedTaskQueue.Len() + taskSchedule.Len()
			case t := <-addToSchedule:
				t.addToSchedule()
			case <-waitUntilNextScheduledTask():
				e := taskSchedule.Front()
				t := e.Value.(*Task)
				t.addToPrioritizedQueue()
				taskSchedule.Remove(e)
			}
		}
	}()

}
