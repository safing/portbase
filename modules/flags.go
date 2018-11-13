package modules

import "flag"

var (
	helpFlag bool
)

func init() {
	flag.BoolVar(&helpFlag, "help", false, "print help")
}

func parseFlags() error {

	// parse flags
	flag.Parse()

	if helpFlag {
		flag.Usage()
		return ErrCleanExit
	}

	return nil
}
