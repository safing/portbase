package modules

import (
	"context"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var (
	mtTestName = "microtask test"
	mtModule   = initNewModule("microtask test module", nil, nil, nil)
)

func init() {
	go microTaskScheduler()
}

func TestMicroTaskWaiting(t *testing.T) { //nolint:paralleltest // Too much interference expected.

	// Check if the state is clean.
	if atomic.LoadInt32(microTasks) != 0 {
		t.Fatalf("cannot start test with dirty state: %d microtasks", atomic.LoadInt32(microTasks))
	}

	// skip
	if testing.Short() {
		t.Skip("skipping test in short mode, as it is not fully deterministic")
	}

	// init
	mtwWaitGroup := new(sync.WaitGroup)
	mtwOutputChannel := make(chan string, 100)
	mtwExpectedOutput := "1234567"
	mtwSleepDuration := 10 * time.Millisecond

	// TEST
	mtwWaitGroup.Add(4)

	// ensure we only execute one microtask at once
	atomic.StoreInt32(microTasksThreshhold, 1)

	// High Priority - slot 1-5
	go func() {
		defer mtwWaitGroup.Done()
		// exec at slot 1
		_ = mtModule.RunHighPriorityMicroTask(mtTestName, func(ctx context.Context) error {
			mtwOutputChannel <- "1" // slot 1
			time.Sleep(mtwSleepDuration * 5)
			mtwOutputChannel <- "2" // slot 5
			return nil
		})
	}()

	time.Sleep(mtwSleepDuration * 1)

	// clear clearances
	_ = mtModule.RunHighPriorityMicroTask(mtTestName, func(ctx context.Context) error {
		return nil
	})

	// Low Priority - slot 16
	go func() {
		defer mtwWaitGroup.Done()
		// exec at slot 2
		_ = mtModule.RunLowPriorityMicroTask(mtTestName, 0, func(ctx context.Context) error {
			mtwOutputChannel <- "7" // slot 16
			return nil
		})
	}()

	time.Sleep(mtwSleepDuration * 1)

	// High Priority - slot 10-15
	go func() {
		defer mtwWaitGroup.Done()
		time.Sleep(mtwSleepDuration * 8)
		// exec at slot 10
		_ = mtModule.RunHighPriorityMicroTask(mtTestName, func(ctx context.Context) error {
			mtwOutputChannel <- "4" // slot 10
			time.Sleep(mtwSleepDuration * 5)
			mtwOutputChannel <- "6" // slot 15
			return nil
		})
	}()

	// Medium Priority - slot 6-13
	go func() {
		defer mtwWaitGroup.Done()
		// exec at slot 3
		_ = mtModule.RunMicroTask(mtTestName, 0, func(ctx context.Context) error {
			mtwOutputChannel <- "3" // slot 6
			time.Sleep(mtwSleepDuration * 7)
			mtwOutputChannel <- "5" // slot 13
			return nil
		})
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
	t.Logf("microTask wait order: %s", completeOutput)
	if completeOutput != mtwExpectedOutput {
		t.Errorf("MicroTask waiting test failed, expected sequence %s, got %s", mtwExpectedOutput, completeOutput)
	}

	// Check if the state is clean.
	time.Sleep(10 * time.Millisecond)
	if atomic.LoadInt32(microTasks) != 0 {
		t.Fatalf("test ends with dirty state: %d microtasks", atomic.LoadInt32(microTasks))
	}
}

// Test Microtask ordering.

// Microtask test globals.

var (
	mtoWaitGroup     sync.WaitGroup
	mtoOutputChannel chan string
	mtoWaitCh        chan struct{}
)

// Microtask test functions.

func highPrioTaskTester() {
	defer mtoWaitGroup.Done()
	<-mtoWaitCh
	_ = mtModule.RunHighPriorityMicroTask(mtTestName, func(ctx context.Context) error {
		mtoOutputChannel <- "0"
		time.Sleep(2 * time.Millisecond)
		return nil
	})
}

func highPrioSignalledTaskTester() {
	defer mtoWaitGroup.Done()
	<-mtoWaitCh
	go func() {
		done := mtModule.SignalHighPriorityMicroTask()
		defer done()

		mtoOutputChannel <- "0"
		time.Sleep(2 * time.Millisecond)
	}()
}

func mediumPrioTaskTester() {
	defer mtoWaitGroup.Done()
	<-mtoWaitCh
	_ = mtModule.RunMicroTask(mtTestName, 0, func(ctx context.Context) error {
		mtoOutputChannel <- "1"
		time.Sleep(2 * time.Millisecond)
		return nil
	})
}

func mediumPrioSignalledTaskTester() {
	defer mtoWaitGroup.Done()
	<-mtoWaitCh
	go func() {
		done := mtModule.SignalMicroTask(0)
		defer done()

		mtoOutputChannel <- "1"
		time.Sleep(2 * time.Millisecond)
	}()
}

func lowPrioTaskTester() {
	defer mtoWaitGroup.Done()
	<-mtoWaitCh
	_ = mtModule.RunLowPriorityMicroTask(mtTestName, 0, func(ctx context.Context) error {
		mtoOutputChannel <- "2"
		time.Sleep(2 * time.Millisecond)
		return nil
	})
}

func lowPrioSignalledTaskTester() {
	defer mtoWaitGroup.Done()
	<-mtoWaitCh
	go func() {
		done := mtModule.SignalLowPriorityMicroTask(0)
		defer done()

		mtoOutputChannel <- "2"
		time.Sleep(2 * time.Millisecond)
	}()
}

func TestMicroTaskOrdering(t *testing.T) { //nolint:paralleltest // Too much interference expected.

	// Check if the state is clean.
	if atomic.LoadInt32(microTasks) != 0 {
		t.Fatalf("cannot start test with dirty state: %d microtasks", atomic.LoadInt32(microTasks))
	}

	// skip
	if testing.Short() {
		t.Skip("skipping test in short mode, as it is not fully deterministic")
	}

	// Only allow a single concurrent task for testing.
	atomic.StoreInt32(microTasksThreshhold, 1)
	defer SetMaxConcurrentMicroTasks(runtime.GOMAXPROCS(0))

	// init
	mtoOutputChannel = make(chan string, 100)
	mtoWaitCh = make(chan struct{})

	// TEST

	// init all in waiting state
	for i := 0; i < 5; i++ {
		mtoWaitGroup.Add(6)
		go lowPrioTaskTester()
		go lowPrioSignalledTaskTester()
		go mediumPrioTaskTester()
		go mediumPrioSignalledTaskTester()
		go highPrioTaskTester()
		go highPrioSignalledTaskTester()
	}

	// wait for all goroutines to be ready
	time.Sleep(10 * time.Millisecond)

	// sync all goroutines
	close(mtoWaitCh)
	// trigger
	select {
	case microTaskFinished <- struct{}{}:
	default:
	}

	// wait for test to finish
	mtoWaitGroup.Wait()

	// collect output
	close(mtoOutputChannel)
	completeOutput := ""
	for s := range mtoOutputChannel {
		completeOutput += s
	}

	// check if test succeeded
	t.Logf("microTask exec order: %s", completeOutput)
	if !strings.Contains(completeOutput, "000") ||
		!strings.Contains(completeOutput, "1111") ||
		!strings.Contains(completeOutput, "22222") {
		t.Errorf("MicroTask ordering test failed, output was %s. This happens occasionally, please run the test multiple times to verify", completeOutput)
	}

	// Check if the state is clean.
	time.Sleep(10 * time.Millisecond)
	if atomic.LoadInt32(microTasks) != 0 {
		t.Fatalf("test ends with dirty state: %d microtasks", atomic.LoadInt32(microTasks))
	}
}
