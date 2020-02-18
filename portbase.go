package main

import (
	"os"

	"github.com/safing/portbase/info"
	"github.com/safing/portbase/run"

	// include packages here
	_ "github.com/safing/portbase/api"
)

func main() {
	// Set Info
	info.Set("Portbase", "0.0.1", "GPLv3", false)

	// Run
	os.Exit(run.Run())
}
