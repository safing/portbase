package info

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Safing/portbase/modules"
)

var (
	showVersion bool
)

func init() {
	modules.Register("info", prep, start, stop)

	flag.BoolVar(&showVersion, "version", false, "show version and exit")
}

func prep() error {
	if !strings.HasSuffix(os.Args[0], ".test") {
		if name == "[NAME]" ||
			version == "[version unknown]" ||
			commit == "[commit unknown]" ||
			buildOptions == "[options unknown]" ||
			buildUser == "[user unknown]" ||
			buildHost == "[host unknown]" ||
			buildDate == "[date unknown]" ||
			buildSource == "[source unknown]" {
			return errors.New("please build using the supplied build script.\n$ ./build {main.go|...}")
		}
	}

	if showVersion {
		fmt.Println(FullVersion())
		return modules.ErrCleanExit
	}
	return nil
}

func start() error {
	return nil
}

func stop() error {
	return nil
}
