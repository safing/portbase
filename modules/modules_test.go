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

func testPrep(t *testing.T, name string) func() error {
	return func() error {
		t.Logf("prep %s\n", name)
		return nil
	}
}

func testStart(t *testing.T, name string) func() error {
	return func() error {
		orderLock.Lock()
		defer orderLock.Unlock()
		t.Logf("start %s\n", name)
		startOrder = fmt.Sprintf("%s>%s", startOrder, name)
		return nil
	}
}

func testStop(t *testing.T, name string) func() error {
	return func() error {
		orderLock.Lock()
		defer orderLock.Unlock()
		t.Logf("stop %s\n", name)
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

func TestModules(t *testing.T) {
	t.Parallel() // Not really, just a workaround for running these tests last.

	t.Run("TestModuleOrder", testModuleOrder)
	t.Run("TestModuleErrors", testModuleErrors)
}

func testModuleOrder(t *testing.T) {

	Register("database", testPrep(t, "database"), testStart(t, "database"), testStop(t, "database"))
	Register("stats", testPrep(t, "stats"), testStart(t, "stats"), testStop(t, "stats"), "database")
	Register("service", testPrep(t, "service"), testStart(t, "service"), testStop(t, "service"), "database")
	Register("analytics", testPrep(t, "analytics"), testStart(t, "analytics"), testStop(t, "analytics"), "stats", "database")

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

func testModuleErrors(t *testing.T) {

	// reset modules
	modules = make(map[string]*Module)
	startComplete.UnSet()
	startCompleteSignal = make(chan struct{})

	// test prep error
	Register("prepfail", testFail, testStart(t, "prepfail"), testStop(t, "prepfail"))
	err := Start()
	if err == nil {
		t.Error("should fail")
	}

	// reset modules
	modules = make(map[string]*Module)
	startComplete.UnSet()
	startCompleteSignal = make(chan struct{})

	// test prep clean exit
	Register("prepcleanexit", testCleanExit, testStart(t, "prepcleanexit"), testStop(t, "prepcleanexit"))
	err = Start()
	if err != ErrCleanExit {
		t.Error("should fail with clean exit")
	}

	// reset modules
	modules = make(map[string]*Module)
	startComplete.UnSet()
	startCompleteSignal = make(chan struct{})

	// test invalid dependency
	Register("database", nil, testStart(t, "database"), testStop(t, "database"), "invalid")
	err = Start()
	if err == nil {
		t.Error("should fail")
	}

	// reset modules
	modules = make(map[string]*Module)
	startComplete.UnSet()
	startCompleteSignal = make(chan struct{})

	// test dependency loop
	Register("database", nil, testStart(t, "database"), testStop(t, "database"), "helper")
	Register("helper", nil, testStart(t, "helper"), testStop(t, "helper"), "database")
	err = Start()
	if err == nil {
		t.Error("should fail")
	}

	// reset modules
	modules = make(map[string]*Module)
	startComplete.UnSet()
	startCompleteSignal = make(chan struct{})

	// test failing module start
	Register("startfail", nil, testFail, testStop(t, "startfail"))
	err = Start()
	if err == nil {
		t.Error("should fail")
	}

	// reset modules
	modules = make(map[string]*Module)
	startComplete.UnSet()
	startCompleteSignal = make(chan struct{})

	// test failing module stop
	Register("stopfail", nil, testStart(t, "stopfail"), testFail)
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
	HelpFlag = true
	err = Start()
	if err == nil {
		t.Error("should fail")
	}
	HelpFlag = false

}
