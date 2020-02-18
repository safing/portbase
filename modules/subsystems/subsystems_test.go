package subsystems

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/safing/portbase/config"
	"github.com/safing/portbase/dataroot"
	"github.com/safing/portbase/modules"
)

func TestSubsystems(t *testing.T) {
	// tmp dir for data root (db & config)
	tmpDir, err := ioutil.TempDir("", "portbase-testing-")
	// initialize data dir
	if err == nil {
		err = dataroot.Initialize(tmpDir, 0755)
	}
	// handle setup error
	if err != nil {
		t.Fatal(err)
	}

	// register

	baseModule := modules.Register("base", nil, nil, nil)
	err = Register(
		"Base",
		"Framework Groundwork",
		baseModule,
		"config:base",
		nil,
	)
	if err != nil {
		t.Fatal(err)
		return
	}

	feature1 := modules.Register("feature1", nil, nil, nil)
	err = Register(
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
	if err != nil {
		t.Fatal(err)
		return
	}
	sub1 := subsystemsMap["Feature One"]

	feature2 := modules.Register("feature2", nil, nil, nil)
	err = Register(
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
	if err != nil {
		t.Fatal(err)
		return
	}

	// start
	err = modules.Start()
	if err != nil {
		t.Fatal(err)
	}

	// test

	// let module fail
	feature1.Error("test-fail", "Testing Fail")
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
	os.RemoveAll(tmpDir)
}
