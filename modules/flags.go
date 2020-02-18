package modules

import "flag"

var (
	// HelpFlag triggers printing flag.Usage. It's exported for custom help handling.
	HelpFlag bool
)

func init() {
	flag.BoolVar(&HelpFlag, "help", false, "print help")
}

func parseFlags() error {
	// parse flags
	flag.Parse()

	if HelpFlag {
		flag.Usage()
		return ErrCleanExit
	}

	return nil
}
