package info

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/safing/portbase/modules"
)

var (
	showVersion bool
)

func init() {
	modules.Register("info", prep, nil, nil)

	flag.BoolVar(&showVersion, "version", false, "show version and exit")
}

func prep() error {
	err := CheckVersion()
	if err != nil {
		return err
	}

	if PrintVersion() {
		return modules.ErrCleanExit
	}
	return nil
}

// CheckVersion checks if the metadata is ok.
func CheckVersion() error {
	if !strings.HasSuffix(os.Args[0], ".test") {
		if name == "[NAME]" ||
			version == "[version unknown]" ||
			commit == "[commit unknown]" ||
			license == "[license unknown]" ||
			buildOptions == "[options unknown]" ||
			buildUser == "[user unknown]" ||
			buildHost == "[host unknown]" ||
			buildDate == "[date unknown]" ||
			buildSource == "[source unknown]" {
			return errors.New("please build using the supplied build script.\n$ ./build {main.go|...}")
		}
	}
	return nil
}

// PrintVersion prints the version, if requested, and returns if it did so.
func PrintVersion() (printed bool) {
	if showVersion {
		fmt.Println(FullVersion())
		return true
	}
	return false
}
