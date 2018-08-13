// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package taskmanager

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// test waiting
func TestMicroTaskWaiting(t *testing.T) {

	// skip
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	// init
	mtwWaitGroup := new(sync.WaitGroup)
	mtwOutputChannel := make(chan string, 100)
	mtwExpectedOutput := "123456"
	mtwSleepDuration := 10 * time.Millisecond

	// TEST
	mtwWaitGroup.Add(3)

	// High Priority - slot 1-5
	go func() {
		defer mtwWaitGroup.Done()
		StartMicroTask()
		mtwOutputChannel <- "1"
		time.Sleep(mtwSleepDuration * 5)
		mtwOutputChannel <- "2"
		EndMicroTask()
	}()

	time.Sleep(mtwSleepDuration * 2)

	// High Priority - slot 10-15
	go func() {
		defer mtwWaitGroup.Done()
		time.Sleep(mtwSleepDuration * 8)
		StartMicroTask()
		mtwOutputChannel <- "4"
		time.Sleep(mtwSleepDuration * 5)
		mtwOutputChannel <- "6"
		EndMicroTask()
	}()

	// Medium Priority - Waits at slot 3, should execute in slot 6-13
	go func() {
		defer mtwWaitGroup.Done()
		<-StartMediumPriorityMicroTask()
		mtwOutputChannel <- "3"
		time.Sleep(mtwSleepDuration * 7)
		mtwOutputChannel <- "5"
		EndMicroTask()
	}()

	// wait for test to finish
	mtwWaitGroup.Wait()

	// collect output
	close(mtwOutputChannel)
	completeOutput := ""
	for s := <-mtwOutputChannel; s != ""; s = <-mtwOutputChannel {
		completeOutput += s
	}
	// check if test succeeded
	if completeOutput != mtwExpectedOutput {
		t.Errorf("MicroTask waiting test failed, expected sequence %s, got %s", mtwExpectedOutput, completeOutput)
	}

}

// test ordering

// globals
var mtoWaitGroup sync.WaitGroup
var mtoOutputChannel chan string
var mtoWaitCh chan bool

// functions
func mediumPrioTaskTester() {
	defer mtoWaitGroup.Done()
	<-mtoWaitCh
	<-StartMediumPriorityMicroTask()
	mtoOutputChannel <- "1"
	time.Sleep(2 * time.Millisecond)
	EndMicroTask()
}

func lowPrioTaskTester() {
	defer mtoWaitGroup.Done()
	<-mtoWaitCh
	<-StartLowPriorityMicroTask()
	mtoOutputChannel <- "2"
	time.Sleep(2 * time.Millisecond)
	EndMicroTask()
}

func veryLowPrioTaskTester() {
	defer mtoWaitGroup.Done()
	<-mtoWaitCh
	<-StartVeryLowPriorityMicroTask()
	mtoOutputChannel <- "3"
	time.Sleep(2 * time.Millisecond)
	EndMicroTask()
}

// test
func TestMicroTaskOrdering(t *testing.T) {

	// skip
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	// init
	mtoOutputChannel = make(chan string, 100)
	mtoWaitCh = make(chan bool, 0)

	// TEST
	mtoWaitGroup.Add(30)

	// kick off
	go mediumPrioTaskTester()
	go mediumPrioTaskTester()
	go lowPrioTaskTester()
	go lowPrioTaskTester()
	go veryLowPrioTaskTester()
	go veryLowPrioTaskTester()
	go lowPrioTaskTester()
	go veryLowPrioTaskTester()
	go mediumPrioTaskTester()
	go veryLowPrioTaskTester()
	go lowPrioTaskTester()
	go mediumPrioTaskTester()
	go veryLowPrioTaskTester()
	go mediumPrioTaskTester()
	go mediumPrioTaskTester()
	go lowPrioTaskTester()
	go mediumPrioTaskTester()
	go lowPrioTaskTester()
	go mediumPrioTaskTester()
	go veryLowPrioTaskTester()
	go veryLowPrioTaskTester()
	go lowPrioTaskTester()
	go mediumPrioTaskTester()
	go veryLowPrioTaskTester()
	go lowPrioTaskTester()
	go lowPrioTaskTester()
	go mediumPrioTaskTester()
	go veryLowPrioTaskTester()
	go lowPrioTaskTester()
	go veryLowPrioTaskTester()

	// wait for all goroutines to be ready
	time.Sleep(10 * time.Millisecond)

	// sync all goroutines
	close(mtoWaitCh)

	// wait for test to finish
	mtoWaitGroup.Wait()

	// collect output
	close(mtoOutputChannel)
	completeOutput := ""
	for s := <-mtoOutputChannel; s != ""; s = <-mtoOutputChannel {
		completeOutput += s
	}
	// check if test succeeded
	if !strings.Contains(completeOutput, "11111") || !strings.Contains(completeOutput, "22222") || !strings.Contains(completeOutput, "33333") {
		t.Errorf("MicroTask ordering test failed, output was %s. This happens occasionally, please run the test multiple times to verify", completeOutput)
	}

}
