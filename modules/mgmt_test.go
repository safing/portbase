package modules

import (
	"testing"
)

func testModuleMgmt(t *testing.T) {
	// enable module management
	EnableModuleManagement(nil)

	registerTestModule(t, "base")
	registerTestModule(t, "feature1", "base")
	registerTestModule(t, "base2", "base")
	registerTestModule(t, "feature2", "base2")
	registerTestModule(t, "sub-feature", "base")
	registerTestModule(t, "feature3", "sub-feature")
	registerTestModule(t, "feature4", "sub-feature")

	// enable core module
	core := modules["base"]
	core.Enable()

	// start and check order
	err := Start()
	if err != nil {
		t.Error(err)
	}
	if changeHistory != " on:base" {
		t.Errorf("order mismatch, was %s", changeHistory)
	}
	changeHistory = ""

	// enable feature1
	feature1 := modules["feature1"]
	feature1.Enable()
	// manage modules and check
	err = ManageModules()
	if err != nil {
		t.Fatal(err)
		return
	}
	if changeHistory != " on:feature1" {
		t.Errorf("order mismatch, was %s", changeHistory)
	}
	changeHistory = ""

	// enable feature2
	feature2 := modules["feature2"]
	feature2.Enable()
	// manage modules and check
	err = ManageModules()
	if err != nil {
		t.Fatal(err)
		return
	}
	if changeHistory != " on:base2 on:feature2" {
		t.Errorf("order mismatch, was %s", changeHistory)
	}
	changeHistory = ""

	// enable feature3
	feature3 := modules["feature3"]
	feature3.Enable()
	// manage modules and check
	err = ManageModules()
	if err != nil {
		t.Fatal(err)
		return
	}
	if changeHistory != " on:sub-feature on:feature3" {
		t.Errorf("order mismatch, was %s", changeHistory)
	}
	changeHistory = ""

	// enable feature4
	feature4 := modules["feature4"]
	feature4.Enable()
	// manage modules and check
	err = ManageModules()
	if err != nil {
		t.Fatal(err)
		return
	}
	if changeHistory != " on:feature4" {
		t.Errorf("order mismatch, was %s", changeHistory)
	}
	changeHistory = ""

	// disable feature1
	feature1.Disable()
	// manage modules and check
	err = ManageModules()
	if err != nil {
		t.Fatal(err)
		return
	}
	if changeHistory != " off:feature1" {
		t.Errorf("order mismatch, was %s", changeHistory)
	}
	changeHistory = ""

	// disable feature3
	feature3.Disable()
	// manage modules and check
	err = ManageModules()
	if err != nil {
		t.Fatal(err)
		return
	}
	// disable feature4
	feature4.Disable()
	// manage modules and check
	err = ManageModules()
	if err != nil {
		t.Fatal(err)
		return
	}
	if changeHistory != " off:feature3 off:feature4 off:sub-feature" {
		t.Errorf("order mismatch, was %s", changeHistory)
	}
	changeHistory = ""

	// enable feature4
	feature4.Enable()
	// manage modules and check
	err = ManageModules()
	if err != nil {
		t.Fatal(err)
		return
	}
	if changeHistory != " on:sub-feature on:feature4" {
		t.Errorf("order mismatch, was %s", changeHistory)
	}
	changeHistory = ""

	// disable feature4
	feature4.Disable()
	// manage modules and check
	err = ManageModules()
	if err != nil {
		t.Fatal(err)
		return
	}
	if changeHistory != " off:feature4 off:sub-feature" {
		t.Errorf("order mismatch, was %s", changeHistory)
	}
	changeHistory = ""

	err = Shutdown()
	if err != nil {
		t.Error(err)
	}
	if changeHistory != " off:feature2 off:base2 off:base" {
		t.Errorf("order mismatch, was %s", changeHistory)
	}

	// reset history
	changeHistory = ""

	// disable module management
	moduleMgmtEnabled.UnSet()

	resetTestEnvironment()
}
