package modules

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/safing/portbase/log"
)

// Default Worker Configuration.
const (
	DefaultBackoffDuration = 2 * time.Second
)

var (
	// ErrRestartNow may be returned (wrapped) by service workers to request an immediate restart.
	ErrRestartNow = errors.New("requested restart")
	errNoModule   = errors.New("missing module (is nil!)")
)

// StartWorker directly starts a generic worker that does not fit to be a Task or MicroTask, such as long running (and possibly mostly idle) sessions. A call to StartWorker starts a new goroutine and returns immediately.
func (m *Module) StartWorker(name string, fn func(context.Context) error) {
	go func() {
		err := m.RunWorker(name, fn)
		switch {
		case err == nil:
			return
		case errors.Is(err, context.Canceled):
			log.Debugf("%s: worker %s was canceled: %s", m.Name, name, err)
		default:
			log.Errorf("%s: worker %s failed: %s", m.Name, name, err)
		}
	}()
}

// RunWorker directly runs a generic worker that does not fit to be a Task or MicroTask, such as long running (and possibly mostly idle) sessions. A call to RunWorker blocks until the worker is finished.
func (m *Module) RunWorker(name string, fn func(context.Context) error) error {
	if m == nil {
		log.Errorf(`modules: cannot start worker "%s" with nil module`, name)
		return errNoModule
	}

	atomic.AddInt32(m.workerCnt, 1)
	defer func() {
		atomic.AddInt32(m.workerCnt, -1)
		m.checkIfStopComplete()
	}()

	return m.runWorker(name, fn)
}

// StartServiceWorker starts a generic worker, which is automatically restarted in case of an error. A call to StartServiceWorker runs the service-worker in a new goroutine and returns immediately. `backoffDuration` specifies how to long to wait before restarts, multiplied by the number of failed attempts. Pass `0` for the default backoff duration. For custom error remediation functionality, build your own error handling procedure using calls to RunWorker.
// Returning nil error or context.Canceled will stop the service worker.
func (m *Module) StartServiceWorker(name string, backoffDuration time.Duration, fn func(context.Context) error) {
	if m == nil {
		log.Errorf(`modules: cannot start service worker "%s" with nil module`, name)
		return
	}

	go m.runServiceWorker(name, backoffDuration, fn)
}

func (m *Module) runServiceWorker(name string, backoffDuration time.Duration, fn func(context.Context) error) {
	atomic.AddInt32(m.workerCnt, 1)
	defer func() {
		atomic.AddInt32(m.workerCnt, -1)
		m.checkIfStopComplete()
	}()

	if backoffDuration == 0 {
		backoffDuration = DefaultBackoffDuration
	}
	failCnt := 0
	lastFail := time.Now()

	for {
		if m.IsStopping() {
			return
		}

		err := m.runWorker(name, fn)
		switch {
		case err == nil:
			// No error means that the worker is finished.
			return

		case errors.Is(err, context.Canceled):
			// A canceled context also means that the worker is finished.
			return

		case errors.Is(err, ErrRestartNow):
			// Worker requested a restart - silently continue with loop.

		default:
			// Any other errors triggers a restart with backoff.

			// Reset fail counter if running without error for some time.
			if time.Now().Add(-5 * time.Minute).After(lastFail) {
				failCnt = 0
			}
			// Increase fail counter and set last failed time.
			failCnt++
			lastFail = time.Now()
			// Log error and back off for some time.
			sleepFor := time.Duration(failCnt) * backoffDuration
			log.Errorf("%s: service-worker %s failed (%d): %s - restarting in %s", m.Name, name, failCnt, err, sleepFor)
			select {
			case <-time.After(sleepFor):
			case <-m.Ctx.Done():
				return
			}
		}
	}
}

func (m *Module) runWorker(name string, fn func(context.Context) error) (err error) {
	defer func() {
		// recover from panic
		panicVal := recover()
		if panicVal != nil {
			me := m.NewPanicError(name, "worker", panicVal)
			me.Report()
			err = me
		}
	}()

	// run
	// TODO: get cancel func for worker context and cancel when worker is done.
	// This ensure that when the worker passes its context to another (async) function, it will also be shutdown when the worker finished or dies.
	err = fn(m.Ctx)
	return
}

func (m *Module) runCtrlFnWithTimeout(name string, timeout time.Duration, fn func() error) error {
	stopFnError := make(chan error)
	go func() {
		stopFnError <- m.runCtrlFn(name, fn)
	}()

	// wait for results
	select {
	case err := <-stopFnError:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("timed out (%s)", timeout)
	}
}

func (m *Module) runCtrlFn(name string, fn func() error) (err error) {
	if fn == nil {
		return
	}

	if m.ctrlFuncRunning.SetToIf(false, true) {
		defer m.ctrlFuncRunning.SetToIf(true, false)
	}

	defer func() {
		// recover from panic
		panicVal := recover()
		if panicVal != nil {
			me := m.NewPanicError(name, "module-control", panicVal)
			me.Report()
			err = me
		}
	}()

	// run
	err = fn()
	return
}
