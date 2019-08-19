package taskmanager

import (
	"sync"
	"testing"
	"time"
)

// test waiting

// globals
var qtWg sync.WaitGroup
var qtOutputChannel chan string
var qtSleepDuration time.Duration

// functions
func queuedTaskTester(s string) {
	t := NewQueuedTask(s)
	go func() {
		<-t.WaitForStart()
		time.Sleep(qtSleepDuration * 2)
		qtOutputChannel <- s
		t.Done()
		qtWg.Done()
	}()
}

func prioritizedTastTester(s string) {
	t := NewPrioritizedQueuedTask(s)
	go func() {
		<-t.WaitForStart()
		time.Sleep(qtSleepDuration * 2)
		qtOutputChannel <- s
		t.Done()
		qtWg.Done()
	}()
}

// test
func TestQueuedTask(t *testing.T) {

	// skip
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	// init
	expectedOutput := "0123456789"
	qtSleepDuration = 10 * time.Millisecond
	qtOutputChannel = make(chan string, 100)
	qtWg.Add(10)

	// test queue length
	c := TotalQueuedTasks()
	if c != 0 {
		t.Errorf("Error in calculating Task Queue, expected 0, got %d", c)
	}

	// TEST
	queuedTaskTester("0")
	queuedTaskTester("1")
	queuedTaskTester("3")
	queuedTaskTester("4")
	queuedTaskTester("6")
	queuedTaskTester("7")
	queuedTaskTester("9")

	// test queue length
	c = TotalQueuedTasks()
	if c != 7 {
		t.Errorf("Error in calculating Task Queue, expected 7, got %d", c)
	}

	time.Sleep(qtSleepDuration * 3)
	prioritizedTastTester("2")
	time.Sleep(qtSleepDuration * 6)
	prioritizedTastTester("5")
	time.Sleep(qtSleepDuration * 6)
	prioritizedTastTester("8")

	// test queue length
	c = TotalQueuedTasks()
	if c != 3 {
		t.Errorf("Error in calculating Task Queue, expected 3, got %d", c)
	}

	// time.Sleep(qtSleepDuration * 100)
	// panic("")

	// wait for test to finish
	qtWg.Wait()

	// test queue length
	c = TotalQueuedTasks()
	if c != 0 {
		t.Errorf("Error in calculating Task Queue, expected 0, got %d", c)
	}

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
