package modules

import (
	"context"
	"fmt"
	"os"
	"runtime/pprof"
	"sync"
	"testing"
	"time"
)

func init() {
	go taskQueueHandler()
	go taskScheduleHandler()

	go func() {
		<-time.After(10 * time.Second)
		fmt.Fprintln(os.Stderr, "taking too long")
		_ = pprof.Lookup("goroutine").WriteTo(os.Stderr, 2)
		os.Exit(1)
	}()

	// always trigger task timeslot for testing
	go func() {
		for {
			taskTimeslot <- struct{}{}
		}
	}()
}

// test waiting

// globals
var qtWg sync.WaitGroup
var qtOutputChannel chan string
var qtSleepDuration time.Duration
var qtModule = initNewModule("task test module", nil, nil, nil)

// functions
func queuedTaskTester(s string) {
	qtModule.NewTask(s, func(ctx context.Context, t *Task) {
		time.Sleep(qtSleepDuration * 2)
		qtOutputChannel <- s
		qtWg.Done()
	}).Queue()
}

func prioritizedTaskTester(s string) {
	qtModule.NewTask(s, func(ctx context.Context, t *Task) {
		time.Sleep(qtSleepDuration * 2)
		qtOutputChannel <- s
		qtWg.Done()
	}).Prioritize()
}

// test
func TestQueuedTask(t *testing.T) {
	// skip
	if testing.Short() {
		t.Skip("skipping test in short mode, as it is not fully deterministic")
	}

	// init
	expectedOutput := "0123456789"
	qtSleepDuration = 20 * time.Millisecond
	qtOutputChannel = make(chan string, 100)
	qtWg.Add(10)

	// TEST
	queuedTaskTester("0")
	queuedTaskTester("1")
	queuedTaskTester("3")
	queuedTaskTester("4")
	queuedTaskTester("6")
	queuedTaskTester("7")
	queuedTaskTester("9")

	time.Sleep(qtSleepDuration * 3)
	prioritizedTaskTester("2")
	time.Sleep(qtSleepDuration * 6)
	prioritizedTaskTester("5")
	time.Sleep(qtSleepDuration * 6)
	prioritizedTaskTester("8")

	// wait for test to finish
	qtWg.Wait()

	// collect output
	close(qtOutputChannel)
	completeOutput := ""
	for s := <-qtOutputChannel; s != ""; s = <-qtOutputChannel {
		completeOutput += s
	}
	// check if test succeeded
	if completeOutput != expectedOutput {
		t.Errorf("QueuedTask test failed, expected sequence %s, got %s", expectedOutput, completeOutput)
	}

}

// test scheduled tasks

// globals
var stWg sync.WaitGroup
var stOutputChannel chan string
var stSleepDuration time.Duration
var stWaitCh chan bool

// functions
func scheduledTaskTester(s string, sched time.Time) {
	qtModule.NewTask(s, func(ctx context.Context, t *Task) {
		time.Sleep(stSleepDuration)
		stOutputChannel <- s
		stWg.Done()
	}).Schedule(sched)
}

// test
func TestScheduledTaskWaiting(t *testing.T) {

	// skip
	if testing.Short() {
		t.Skip("skipping test in short mode, as it is not fully deterministic")
	}

	// init
	expectedOutput := "0123456789"
	stSleepDuration = 10 * time.Millisecond
	stOutputChannel = make(chan string, 100)
	stWaitCh = make(chan bool)

	stWg.Add(10)

	// TEST
	scheduledTaskTester("4", time.Now().Add(stSleepDuration*8))
	scheduledTaskTester("0", time.Now().Add(stSleepDuration*0))
	scheduledTaskTester("8", time.Now().Add(stSleepDuration*16))
	scheduledTaskTester("1", time.Now().Add(stSleepDuration*2))
	scheduledTaskTester("7", time.Now().Add(stSleepDuration*14))
	scheduledTaskTester("9", time.Now().Add(stSleepDuration*18))
	scheduledTaskTester("3", time.Now().Add(stSleepDuration*6))
	scheduledTaskTester("2", time.Now().Add(stSleepDuration*4))
	scheduledTaskTester("6", time.Now().Add(stSleepDuration*12))
	scheduledTaskTester("5", time.Now().Add(stSleepDuration*10))

	// wait for test to finish
	close(stWaitCh)
	stWg.Wait()

	// collect output
	close(stOutputChannel)
	completeOutput := ""
	for s := <-stOutputChannel; s != ""; s = <-stOutputChannel {
		completeOutput += s
	}
	// check if test succeeded
	if completeOutput != expectedOutput {
		t.Errorf("ScheduledTask test failed, expected sequence %s, got %s", expectedOutput, completeOutput)
	}

}
