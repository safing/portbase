package modules

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/safing/portbase/log"
)

// Worker Default Configuration
const (
	DefaultBackoffDuration = 2 * time.Second
)

// RunWorker directly runs a generic worker that does not fit to be a Task or MicroTask, such as long running (and possibly mostly idle) sessions. A call to RunWorker blocks until the worker is finished.
func (m *Module) RunWorker(name string, fn func(context.Context) error) error {
	atomic.AddInt32(m.workerCnt, 1)
	m.waitGroup.Add(1)
	defer func() {
		atomic.AddInt32(m.workerCnt, -1)
		m.waitGroup.Done()
	}()

	return m.runWorker(name, fn)
}

// StartServiceWorker starts a generic worker, which is automatically restarted in case of an error. A call to StartServiceWorker runs the service-worker in a new goroutine and returns immediately. `backoffDuration` specifies how to long to wait before restarts, multiplied by the number of failed attempts. Pass `0` for the default backoff duration. For custom error remediation functionality, build your own error handling procedure using calls to RunWorker.
func (m *Module) StartServiceWorker(name string, backoffDuration time.Duration, fn func(context.Context) error) {
	go m.runServiceWorker(name, backoffDuration, fn)
}

func (m *Module) runServiceWorker(name string, backoffDuration time.Duration, fn func(context.Context) error) {
	atomic.AddInt32(m.workerCnt, 1)
	m.waitGroup.Add(1)
	defer func() {
		atomic.AddInt32(m.workerCnt, -1)
		m.waitGroup.Done()
	}()

	if backoffDuration == 0 {
		backoffDuration = DefaultBackoffDuration
	}
	failCnt := 0

	for {
		if m.ShutdownInProgress() {
			return
		}

		err := m.runWorker(name, fn)
		if err != nil {
			// log error and restart
			failCnt++
			sleepFor := time.Duration(failCnt) * backoffDuration
			log.Errorf("%s: service-worker %s failed (%d): %s - restarting in %s", m.Name, name, failCnt, err, sleepFor)
			time.Sleep(sleepFor)
		} else {
			// finish
			return
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
	err = fn(m.Ctx)
	return
}
