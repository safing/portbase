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
		<-time.After(30 * time.Second)
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

// Test queued tasks.

// Queued task test globals.

var (
	qtWg            sync.WaitGroup
	qtOutputChannel chan string
	qtSleepDuration time.Duration
	qtModule        *Module
)

func init() {
	qtModule = initNewModule("task test module", nil, nil, nil)
	qtModule.status = StatusOnline
}

// Queued task test functions.

func queuedTaskTester(s string) {
	qtModule.NewTask(s, func(ctx context.Context, t *Task) error {
		time.Sleep(qtSleepDuration * 2)
		qtOutputChannel <- s
		qtWg.Done()
		return nil
	}).Queue()
}

func prioritizedTaskTester(s string) {
	qtModule.NewTask(s, func(ctx context.Context, t *Task) error {
		time.Sleep(qtSleepDuration * 2)
		qtOutputChannel <- s
		qtWg.Done()
		return nil
	}).QueuePrioritized()
}

func TestQueuedTask(t *testing.T) { //nolint:paralleltest // Too much interference expected.
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

// Test scheduled tasks.

// Scheduled task test globals.

var (
	stWg            sync.WaitGroup
	stOutputChannel chan string
	stSleepDuration time.Duration
	stWaitCh        chan bool
)

// Scheduled task test functions.

func scheduledTaskTester(s string, sched time.Time) {
	qtModule.NewTask(s, func(ctx context.Context, t *Task) error {
		time.Sleep(stSleepDuration)
		stOutputChannel <- s
		stWg.Done()
		return nil
	}).Schedule(sched)
}

func TestScheduledTaskWaiting(t *testing.T) { //nolint:paralleltest // Too much interference expected.

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

func TestRequeueingTask(t *testing.T) { //nolint:paralleltest // Too much interference expected.
	blockWg := &sync.WaitGroup{}
	wg := &sync.WaitGroup{}

	// block task execution
	blockWg.Add(1) // mark done at beginning
	wg.Add(2)      // mark done at end
	block := qtModule.NewTask("TestRequeueingTask:block", func(ctx context.Context, t *Task) error {
		blockWg.Done()
		time.Sleep(100 * time.Millisecond)
		wg.Done()
		return nil
	}).StartASAP()
	// make sure first task has started
	blockWg.Wait()
	// fmt.Printf("%s: %+v\n", time.Now(), block)

	// schedule again while executing
	blockWg.Add(1) // mark done at beginning
	block.StartASAP()
	// fmt.Printf("%s: %+v\n", time.Now(), block)

	// test task
	wg.Add(1)
	task := qtModule.NewTask("TestRequeueingTask:test", func(ctx context.Context, t *Task) error {
		wg.Done()
		return nil
	}).Schedule(time.Now().Add(2 * time.Second))

	// reschedule
	task.Schedule(time.Now().Add(1 * time.Second))
	task.Queue()
	task.QueuePrioritized()
	task.StartASAP()
	wg.Wait()
	time.Sleep(100 * time.Millisecond) // let tasks finalize execution

	// do it again

	// block task execution (while first block task is still running!)
	blockWg.Add(1) // mark done at beginning
	wg.Add(1)      // mark done at end
	block.StartASAP()
	blockWg.Wait()
	// reschedule
	wg.Add(1)
	task.Schedule(time.Now().Add(1 * time.Second))
	task.Queue()
	task.QueuePrioritized()
	task.StartASAP()
	wg.Wait()
}

func TestQueueSuccession(t *testing.T) { //nolint:paralleltest // Too much interference expected.
	var cnt int
	wg := &sync.WaitGroup{}
	wg.Add(10)

	tt := qtModule.NewTask("TestRequeueingTask:test", func(ctx context.Context, task *Task) error {
		time.Sleep(10 * time.Millisecond)
		wg.Done()
		cnt++
		fmt.Printf("completed succession %d\n", cnt)
		switch cnt {
		case 1, 4, 6:
			task.Queue()
		case 2, 5, 8:
			task.StartASAP()
		case 3, 7, 9:
			task.Schedule(time.Now().Add(10 * time.Millisecond))
		}
		return nil
	})
	// fmt.Printf("%+v\n", tt)
	tt.StartASAP()
	// time.Sleep(100 * time.Millisecond)
	// fmt.Printf("%+v\n", tt)

	wg.Wait()
}
