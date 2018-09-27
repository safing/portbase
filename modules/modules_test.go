// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package modules

import (
	"errors"
	"fmt"
	"testing"
	"sync"
	"time"
)

var (
	startOrder    string
	shutdownOrder string
)

func testPrep() error {
	return nil
}

func testStart(name string) func() error {
	return func() error {
		startOrder = fmt.Sprintf("%s>%s", startOrder, name)
		return nil
	}
}

func testStop(name string) func() error {
	return func() error {
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

	Register("database", testPrep, testStart("database"), testStop("database"))
	Register("stats", testPrep, testStart("stats"), testStop("stats"), "database")
	Register("service", testPrep, testStart("service"), testStop("service"), "database")
	Register("analytics", testPrep, testStart("analytics"), testStop("analytics"), "stats", "database")

	Start()

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
	Shutdown()

	if startOrder != ">database>service>stats>analytics" &&
		startOrder != ">database>stats>service>analytics" &&
		startOrder != ">database>stats>analytics>service" {
		t.Errorf("start order mismatch, was %s", startOrder)
	}
	if shutdownOrder != ">analytics>service>stats>database" &&
		shutdownOrder != ">analytics>stats>service>database" &&
		shutdownOrder != ">service>analytics>stats>database" {
		t.Errorf("shutdown order mismatch, was %s", shutdownOrder)
	}

	wg.Wait()
}

func resetModules() {
	for _, module := range modules {
		module.Active.UnSet()
		module.inTransition = false
	}
}

func TestErrors(t *testing.T) {

	// reset modules
	modules = make(map[string]*Module)
	modulesOrder = make([]*Module, 0)
	startComplete.UnSet()

	// test prep error
	Register("prepfail", testFail, testStart("prepfail"), testStop("prepfail"))
	err := Start()
	if err == nil {
		t.Error("should fail")
	}

	// reset modules
	modules = make(map[string]*Module)
	modulesOrder = make([]*Module, 0)
	startComplete.UnSet()

	// test prep clean exit
	Register("prepcleanexit", testCleanExit, testStart("prepcleanexit"), testStop("prepcleanexit"))
	err = Start()
	if err != ErrCleanExit {
		t.Error("should fail with clean exit")
	}

	// reset modules
	modules = make(map[string]*Module)
	modulesOrder = make([]*Module, 0)
	startComplete.UnSet()

	// test invalid dependency
	Register("database", testPrep, testStart("database"), testStop("database"), "invalid")
	// go func() {
	// 	time.Sleep(1 * time.Second)
	// 	fmt.Println("===== TAKING TOO LONG FOR SHUTDOWN - PRINTING STACK TRACES =====")
	// 	pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
	// 	os.Exit(1)
	// }()
	err = Start()
	if err == nil {
		t.Error("should fail")
	}

	// reset modules
	modules = make(map[string]*Module)
	modulesOrder = make([]*Module, 0)
	startComplete.UnSet()

	// test dependency loop
	Register("database", testPrep, testStart("database"), testStop("database"), "helper")
	Register("helper", testPrep, testStart("helper"), testStop("helper"), "database")
	err = Start()
	if err == nil {
		t.Error("should fail")
	}

	// reset modules
	modules = make(map[string]*Module)
	modulesOrder = make([]*Module, 0)
	startComplete.UnSet()

	// test failing module start
	Register("startfail", testPrep, testFail, testStop("startfail"))
	err = Start()
	if err == nil {
		t.Error("should fail")
	}

	// reset modules
	modules = make(map[string]*Module)
	modulesOrder = make([]*Module, 0)
	startComplete.UnSet()

	// test failing module stop
	Register("stopfail", testPrep, testStart("stopfail"), testFail)
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
	modulesOrder = make([]*Module, 0)
	startComplete.UnSet()

	// test help flag
	helpFlag = true
	err = Start()
	if err == nil {
		t.Error("should fail")
	}
	helpFlag = false

}
