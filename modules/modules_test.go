// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package modules

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

var (
	orderLock     sync.Mutex
	startOrder    string
	shutdownOrder string
)

func testPrep(name string) func() error {
	return func() error {
		// fmt.Printf("prep %s\n", name)
		return nil
	}
}

func testStart(name string) func() error {
	return func() error {
		orderLock.Lock()
		defer orderLock.Unlock()
		// fmt.Printf("start %s\n", name)
		startOrder = fmt.Sprintf("%s>%s", startOrder, name)
		return nil
	}
}

func testStop(name string) func() error {
	return func() error {
		orderLock.Lock()
		defer orderLock.Unlock()
		// fmt.Printf("stop %s\n", name)
		shutdownOrder = fmt.Sprintf("%s>%s", shutdownOrder, name)
		return nil
	}
}

func testFail() error {
	return errors.New("test error")
}

func testCleanExit() error {
	return ErrCleanExit
}

func TestOrdering(t *testing.T) {

	Register("database", testPrep("database"), testStart("database"), testStop("database"))
	Register("stats", testPrep("stats"), testStart("stats"), testStop("stats"), "database")
	Register("service", testPrep("service"), testStart("service"), testStop("service"), "database")
	Register("analytics", testPrep("analytics"), testStart("analytics"), testStop("analytics"), "stats", "database")

	err := Start()
	if err != nil {
		t.Error(err)
	}

	if startOrder != ">database>service>stats>analytics" &&
		startOrder != ">database>stats>service>analytics" &&
		startOrder != ">database>stats>analytics>service" {
		t.Errorf("start order mismatch, was %s", startOrder)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		select {
		case <-ShuttingDown():
		case <-time.After(1 * time.Second):
			t.Error("did not receive shutdown signal")
		}
		wg.Done()
	}()
	err = Shutdown()
	if err != nil {
		t.Error(err)
	}

	if shutdownOrder != ">analytics>service>stats>database" &&
		shutdownOrder != ">analytics>stats>service>database" &&
		shutdownOrder != ">service>analytics>stats>database" {
		t.Errorf("shutdown order mismatch, was %s", shutdownOrder)
	}

	wg.Wait()

	printAndRemoveModules()
}

func printAndRemoveModules() {
	modulesLock.Lock()
	defer modulesLock.Unlock()

	fmt.Printf("All %d modules:\n", len(modules))
	for _, m := range modules {
		fmt.Printf("module %s: %+v\n", m.Name, m)
	}

	modules = make(map[string]*Module)
}

func resetModules() {
	for _, module := range modules {
		module.Prepped.UnSet()
		module.Started.UnSet()
		module.Stopped.UnSet()
		module.inTransition.UnSet()

		module.depModules = make([]*Module, 0)
		module.depModules = make([]*Module, 0)
	}
}

func TestErrors(t *testing.T) {

	// reset modules
	modules = make(map[string]*Module)
	startComplete.UnSet()
	startCompleteSignal = make(chan struct{})

	// test prep error
	Register("prepfail", testFail, testStart("prepfail"), testStop("prepfail"))
	err := Start()
	if err == nil {
		t.Error("should fail")
	}

	// reset modules
	modules = make(map[string]*Module)
	startComplete.UnSet()
	startCompleteSignal = make(chan struct{})

	// test prep clean exit
	Register("prepcleanexit", testCleanExit, testStart("prepcleanexit"), testStop("prepcleanexit"))
	err = Start()
	if err != ErrCleanExit {
		t.Error("should fail with clean exit")
	}

	// reset modules
	modules = make(map[string]*Module)
	startComplete.UnSet()
	startCompleteSignal = make(chan struct{})

	// test invalid dependency
	Register("database", nil, testStart("database"), testStop("database"), "invalid")
	err = Start()
	if err == nil {
		t.Error("should fail")
	}

	// reset modules
	modules = make(map[string]*Module)
	startComplete.UnSet()
	startCompleteSignal = make(chan struct{})

	// test dependency loop
	Register("database", nil, testStart("database"), testStop("database"), "helper")
	Register("helper", nil, testStart("helper"), testStop("helper"), "database")
	err = Start()
	if err == nil {
		t.Error("should fail")
	}

	// reset modules
	modules = make(map[string]*Module)
	startComplete.UnSet()
	startCompleteSignal = make(chan struct{})

	// test failing module start
	Register("startfail", nil, testFail, testStop("startfail"))
	err = Start()
	if err == nil {
		t.Error("should fail")
	}

	// reset modules
	modules = make(map[string]*Module)
	startComplete.UnSet()
	startCompleteSignal = make(chan struct{})

	// test failing module stop
	Register("stopfail", nil, testStart("stopfail"), testFail)
	err = Start()
	if err != nil {
		t.Error("should not fail")
	}
	err = Shutdown()
	if err == nil {
		t.Error("should fail")
	}

	// reset modules
	modules = make(map[string]*Module)
	startComplete.UnSet()
	startCompleteSignal = make(chan struct{})

	// test help flag
	helpFlag = true
	err = Start()
	if err == nil {
		t.Error("should fail")
	}
	helpFlag = false

}
