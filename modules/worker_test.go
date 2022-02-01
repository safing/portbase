package modules

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

var (
	wModule = initNewModule("worker test module", nil, nil, nil)
	errTest = errors.New("test error")
)

func TestWorker(t *testing.T) { //nolint:paralleltest // Too much interference expected.
	// test basic functionality
	err := wModule.RunWorker("test worker", func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("worker failed (should not): %s", err)
	}

	// test returning an error
	err = wModule.RunWorker("test worker", func(ctx context.Context) error {
		return errTest
	})
	if !errors.Is(err, errTest) {
		t.Errorf("worker failed with unexpected error: %s", err)
	}

	// test service functionality
	failCnt := 0
	var sWTestGroup sync.WaitGroup
	sWTestGroup.Add(1)
	wModule.StartServiceWorker("test service-worker", 2*time.Millisecond, func(ctx context.Context) error {
		failCnt++
		t.Logf("service-worker test run #%d", failCnt)
		if failCnt >= 3 {
			sWTestGroup.Done()
			return nil
		}
		return errTest
	})
	// wait for service-worker to complete test
	sWTestGroup.Wait()
	if failCnt != 3 {
		t.Errorf("service-worker failed to restart")
	}

	// test panic recovery
	err = wModule.RunWorker("test worker", func(ctx context.Context) error {
		var a []byte
		_ = a[0]
		return nil
	})
	t.Logf("panic error message: %s", err)
	panicked, mErr := IsPanic(err)
	if !panicked {
		t.Errorf("failed to return *ModuleError, got %+v", err)
	} else {
		t.Logf("panic stack trace:\n%s", mErr.StackTrace)
	}
}
