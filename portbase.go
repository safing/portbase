package main

import (
	"os"

	_ "github.com/safing/portbase/api"
	"github.com/safing/portbase/info"
	"github.com/safing/portbase/run"
)

func main() {
	// Set Info
	info.Set("Portbase", "0.0.1", "GPLv3")

	// Run
	os.Exit(run.Run())
}
