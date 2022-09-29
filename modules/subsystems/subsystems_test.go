package subsystems

import (
	"os"
	"testing"
	"time"

	"github.com/safing/portbase/config"
	_ "github.com/safing/portbase/database/dbmodule"
	"github.com/safing/portbase/dataroot"
	"github.com/safing/portbase/modules"
)

func TestSubsystems(t *testing.T) { //nolint:paralleltest // Too much interference expected.
	// tmp dir for data root (db & config)
	tmpDir, err := os.MkdirTemp("", "portbase-testing-")
	// initialize data dir
	if err == nil {
		err = dataroot.Initialize(tmpDir, 0o0755)
	}
	// handle setup error
	if err != nil {
		t.Fatal(err)
	}

	// register

	baseModule := modules.Register("base", nil, nil, nil)
	Register(
		"base",
		"Base",
		"Framework Groundwork",
		baseModule,
		"config:base",
		nil,
	)

	feature1 := modules.Register("feature1", nil, nil, nil)
	Register(
		"feature-one",
		"Feature One",
		"Provides feature one",
		feature1,
		"config:feature1",
		&config.Option{
			Name:         "Enable Feature One",
			Key:          "config:subsystems/feature1",
			Description:  "This option enables feature 1",
			OptType:      config.OptTypeBool,
			DefaultValue: false,
		},
	)
	sub1 := DefaultManager.subsys["feature-one"]

	feature2 := modules.Register("feature2", nil, nil, nil)
	Register(
		"feature-two",
		"Feature Two",
		"Provides feature two",
		feature2,
		"config:feature2",
		&config.Option{
			Name:         "Enable Feature One",
			Key:          "config:subsystems/feature2",
			Description:  "This option enables feature 2",
			OptType:      config.OptTypeBool,
			DefaultValue: false,
		},
	)

	// start
	err = modules.Start()
	if err != nil {
		t.Fatal(err)
	}

	// test

	// let module fail
	feature1.Error("test-fail", "Test Fail", "Testing Fail")
	time.Sleep(10 * time.Millisecond)
	if sub1.FailureStatus != modules.FailureError {
		t.Fatal("error did not propagate")
	}

	// resolve
	feature1.Resolve("test-fail")
	time.Sleep(10 * time.Millisecond)
	if sub1.FailureStatus != modules.FailureNone {
		t.Fatal("error resolving did not propagate")
	}

	// update settings
	err = config.SetConfigOption("config:subsystems/feature2", true)
	if err != nil {
		t.Fatal(err)
		return
	}
	time.Sleep(200 * time.Millisecond)
	if !feature2.Enabled() {
		t.Fatal("failed to enable feature2")
	}
	if feature2.Status() != modules.StatusOnline {
		t.Fatal("feature2 did not start")
	}

	// update settings
	err = config.SetConfigOption("config:subsystems/feature2", false)
	if err != nil {
		t.Fatal(err)
		return
	}
	time.Sleep(200 * time.Millisecond)
	if feature2.Enabled() {
		t.Fatal("failed to disable feature2")
	}
	if feature2.Status() != modules.StatusOffline {
		t.Fatal("feature2 did not stop")
	}

	// clean up and exit
	_ = os.RemoveAll(tmpDir)
}
