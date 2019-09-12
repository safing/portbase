package modules

import (
	"context"
	"errors"
	"testing"
	"time"
)

var (
	wModule = initNewModule("worker test module", nil, nil, nil)
	errTest = errors.New("test error")
)

func TestWorker(t *testing.T) {
	// test basic functionality
	err := wModule.StartWorker("test worker", false, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("worker failed (should not): %s", err)
	}

	// test returning an error
	err = wModule.StartWorker("test worker", false, func(ctx context.Context) error {
		return errTest
	})
	if err != errTest {
		t.Errorf("worker failed with unexpected error: %s", err)
	}

	// test service functionality
	serviceBackoffDuration = 2 * time.Millisecond // speed up backoff
	failCnt := 0
	err = wModule.StartWorker("test worker", true, func(ctx context.Context) error {
		failCnt++
		t.Logf("service-worker test run #%d", failCnt)
		if failCnt >= 3 {
			return nil
		}
		return errTest
	})
	if err == errTest {
		t.Errorf("service-worker failed with unexpected error: %s", err)
	}
	if failCnt != 3 {
		t.Errorf("service-worker failed to restart")
	}

	// test panic recovery
	err = wModule.StartWorker("test worker", false, func(ctx context.Context) error {
		var a []byte
		_ = a[0] //nolint // we want to runtime panic!
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
