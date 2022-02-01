package modules

var cmdLineOperation func() error

// SetCmdLineOperation sets a command line operation to be executed instead of starting the system. This is useful when functions need all modules to be prepared for a special operation.
func SetCmdLineOperation(fn func() error) {
	cmdLineOperation = fn
}
