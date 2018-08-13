// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package taskmanager

import (
	"sync"
	"testing"
	"time"
)

// test waiting

// globals
var stWg sync.WaitGroup
var stOutputChannel chan string
var stSleepDuration time.Duration
var stWaitCh chan bool

// functions
func scheduledTaskTester(s string, sched time.Time) {
	t := NewScheduledTask(s, sched)
	go func() {
		<-stWaitCh
		<-t.WaitForStart()
		time.Sleep(stSleepDuration)
		stOutputChannel <- s
		t.Done()
		stWg.Done()
	}()
}

// test
func TestScheduledTaskWaiting(t *testing.T) {

	// skip
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	// init
	expectedOutput := "0123456789"
	stSleepDuration = 10 * time.Millisecond
	stOutputChannel = make(chan string, 100)
	stWaitCh = make(chan bool, 0)

	// test queue length
	c := TotalScheduledTasks()
	if c != 0 {
		t.Errorf("Error in calculating Task Queue, expected 0, got %d", c)
	}

	stWg.Add(10)

	// TEST
	scheduledTaskTester("4", time.Now().Add(stSleepDuration*4))
	scheduledTaskTester("0", time.Now().Add(stSleepDuration*1))
	scheduledTaskTester("8", time.Now().Add(stSleepDuration*8))
	scheduledTaskTester("1", time.Now().Add(stSleepDuration*2))
	scheduledTaskTester("7", time.Now().Add(stSleepDuration*7))

	// test queue length
	time.Sleep(1 * time.Millisecond)
	c = TotalScheduledTasks()
	if c != 5 {
		t.Errorf("Error in calculating Task Queue, expected 5, got %d", c)
	}

	scheduledTaskTester("9", time.Now().Add(stSleepDuration*9))
	scheduledTaskTester("3", time.Now().Add(stSleepDuration*3))
	scheduledTaskTester("2", time.Now().Add(stSleepDuration*2))
	scheduledTaskTester("6", time.Now().Add(stSleepDuration*6))
	scheduledTaskTester("5", time.Now().Add(stSleepDuration*5))

	// wait for test to finish
	close(stWaitCh)
	stWg.Wait()

	// test queue length
	c = TotalScheduledTasks()
	if c != 0 {
		t.Errorf("Error in calculating Task Queue, expected 0, got %d", c)
	}

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
