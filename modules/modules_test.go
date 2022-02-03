package modules

import (
	"errors"
	"fmt"
	"sync"
	"testing"
)

var (
	changeHistoryLock sync.Mutex
	changeHistory     string
)

func registerTestModule(t *testing.T, name string, dependencies ...string) {
	t.Helper()

	Register(
		name,
		func() error {
			t.Logf("prep %s\n", name)
			return nil
		},
		func() error {
			changeHistoryLock.Lock()
			defer changeHistoryLock.Unlock()
			t.Logf("start %s\n", name)
			changeHistory = fmt.Sprintf("%s on:%s", changeHistory, name)
			return nil
		},
		func() error {
			changeHistoryLock.Lock()
			defer changeHistoryLock.Unlock()
			t.Logf("stop %s\n", name)
			changeHistory = fmt.Sprintf("%s off:%s", changeHistory, name)
			return nil
		},
		dependencies...,
	)
}

func testFail() error {
	return errors.New("test error")
}

func testCleanExit() error {
	return ErrCleanExit
}

func TestModules(t *testing.T) { //nolint:tparallel // Too much interference expected.
	t.Parallel() // Not really, just a workaround for running these tests last.

	t.Run("TestModuleOrder", testModuleOrder)   //nolint:paralleltest // Too much interference expected.
	t.Run("TestModuleMgmt", testModuleMgmt)     //nolint:paralleltest // Too much interference expected.
	t.Run("TestModuleErrors", testModuleErrors) //nolint:paralleltest // Too much interference expected.
}

func testModuleOrder(t *testing.T) {
	registerTestModule(t, "database")
	registerTestModule(t, "stats", "database")
	registerTestModule(t, "service", "database")
	registerTestModule(t, "analytics", "stats", "database")

	err := Start()
	if err != nil {
		t.Error(err)
	}

	if changeHistory != " on:database on:service on:stats on:analytics" &&
		changeHistory != " on:database on:stats on:service on:analytics" &&
		changeHistory != " on:database on:stats on:analytics on:service" {
		t.Errorf("start order mismatch, was %s", changeHistory)
	}
	changeHistory = ""

	err = Shutdown()
	if err != nil {
		t.Error(err)
	}

	if changeHistory != " off:analytics off:service off:stats off:database" &&
		changeHistory != " off:analytics off:stats off:service off:database" &&
		changeHistory != " off:service off:analytics off:stats off:database" {
		t.Errorf("shutdown order mismatch, was %s", changeHistory)
	}
	changeHistory = ""

	resetTestEnvironment()
}

func testModuleErrors(t *testing.T) {
	// test prep error
	Register("prepfail", testFail, nil, nil)
	err := Start()
	if err == nil {
		t.Error("should fail")
	}

	resetTestEnvironment()

	// test prep clean exit
	Register("prepcleanexit", testCleanExit, nil, nil)
	err = Start()
	if !errors.Is(err, ErrCleanExit) {
		t.Error("should fail with clean exit")
	}

	resetTestEnvironment()

	// test invalid dependency
	Register("database", nil, nil, nil, "invalid")
	err = Start()
	if err == nil {
		t.Error("should fail")
	}

	resetTestEnvironment()

	// test dependency loop
	registerTestModule(t, "database", "helper")
	registerTestModule(t, "helper", "database")
	err = Start()
	if err == nil {
		t.Error("should fail")
	}

	resetTestEnvironment()

	// test failing module start
	Register("startfail", nil, testFail, nil)
	err = Start()
	if err == nil {
		t.Error("should fail")
	}

	resetTestEnvironment()

	// test failing module stop
	Register("stopfail", nil, nil, testFail)
	err = Start()
	if err != nil {
		t.Error("should not fail")
	}
	err = Shutdown()
	if err == nil {
		t.Error("should fail")
	}

	resetTestEnvironment()

	// test help flag
	HelpFlag = true
	err = Start()
	if err == nil {
		t.Error("should fail")
	}
	HelpFlag = false

	resetTestEnvironment()
}

func printModules() { //nolint:unused,deadcode
	fmt.Printf("All %d modules:\n", len(modules))
	for _, m := range modules {
		fmt.Printf("module %s: %+v\n", m.Name, m)
	}
}

func resetTestEnvironment() {
	modules = make(map[string]*Module)
	shutdownSignal = make(chan struct{})
	shutdownCompleteSignal = make(chan struct{})
	shutdownFlag.UnSet()
	modulesLocked.UnSet()
}
