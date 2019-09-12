package modules

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/safing/portbase/log"
)

var (
	serviceBackoffDuration = 2 * time.Second
)

// StartWorker starts a generic worker that does not fit to be a Task or MicroTask, such as long running (and possibly mostly idle) sessions. You may declare a worker as a service, which will then be automatically restarted in case of an error.
func (m *Module) StartWorker(name string, service bool, fn func(context.Context) error) error {
	atomic.AddInt32(m.workerCnt, 1)
	m.waitGroup.Add(1)
	defer func() {
		atomic.AddInt32(m.workerCnt, -1)
		m.waitGroup.Done()
	}()

	failCnt := 0

	if service {
		for {
			if m.ShutdownInProgress() {
				return nil
			}

			err := m.runWorker(name, fn)
			if err != nil {
				// log error and restart
				failCnt++
				sleepFor := time.Duration(failCnt) * serviceBackoffDuration
				log.Errorf("module %s service-worker %s failed (%d): %s - restarting in %s", m.Name, name, failCnt, err, sleepFor)
				time.Sleep(sleepFor)
			} else {
				// clean finish
				return nil
			}
		}
	} else {
		return m.runWorker(name, fn)
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
