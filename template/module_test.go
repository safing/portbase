package template

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	_ "github.com/safing/portbase/database/dbmodule"
	"github.com/safing/portbase/dataroot"
	"github.com/safing/portbase/modules"
)

func TestMain(m *testing.M) {
	// register base module, for database initialization
	modules.Register("base", nil, nil, nil)

	// enable module for testing
	module.Enable()

	// tmp dir for data root (db & config)
	tmpDir, err := ioutil.TempDir("", "portbase-testing-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create tmp dir: %s\n", err)
		os.Exit(1)
	}
	// initialize data dir
	err = dataroot.Initialize(tmpDir, 0o0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize data root: %s\n", err)
		os.Exit(1)
	}

	// start modules
	var exitCode int
	err = modules.Start()
	if err != nil {
		// starting failed
		fmt.Fprintf(os.Stderr, "failed to setup test: %s\n", err)
		exitCode = 1
	} else {
		// run tests
		exitCode = m.Run()
	}

	// shutdown
	_ = modules.Shutdown()
	if modules.GetExitStatusCode() != 0 {
		exitCode = modules.GetExitStatusCode()
		fmt.Fprintf(os.Stderr, "failed to cleanly shutdown test: %s\n", err)
	}
	// clean up and exit
	os.RemoveAll(tmpDir)
	os.Exit(exitCode)
}
