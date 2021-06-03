package updater

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/safing/portbase/utils"
)

var (
	registry *ResourceRegistry
)

func TestMain(m *testing.M) {
	// setup
	tmpDir, err := ioutil.TempDir("", "ci-portmaster-")
	if err != nil {
		panic(err)
	}
	registry = &ResourceRegistry{
		UsePreReleases: true,
		DevMode:        true,
		Online:         true,
	}
	err = registry.Initialize(utils.NewDirStructure(tmpDir, 0777))
	if err != nil {
		panic(err)
	}

	// run
	// call flag.Parse() here if TestMain uses flags
	ret := m.Run()

	// teardown
	os.RemoveAll(tmpDir)
	os.Exit(ret)
}
