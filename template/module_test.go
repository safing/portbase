package template

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/safing/portbase/dataroot"
	"github.com/safing/portbase/modules"
)

func TestMain(m *testing.M) {
	// tmp dir for data root (db & config)
	tmpDir, err := ioutil.TempDir("", "portbase-testing-")
	// initialize data dir
	if err == nil {
		err = dataroot.Initialize(tmpDir, 0755)
	}
	// start modules
	if err == nil {
		err = modules.Start()
	}
	// handle setup error
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to setup test: %s", err)
		os.Exit(1)
	}

	// run tests
	exitCode := m.Run()

	// shutdown
	_ = modules.Shutdown()
	if modules.GetExitStatusCode() != 0 {
		exitCode = modules.GetExitStatusCode()
		fmt.Fprintf(os.Stderr, "failed to cleanly shutdown test: %s", err)
	}
	// clean up and exit
	os.RemoveAll(tmpDir)
	os.Exit(exitCode)
}
