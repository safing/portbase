package modules

var (
	exitStatusCode int
)

// SetExitStatusCode sets the exit code that the program shell return to the host after shutdown.
func SetExitStatusCode(n int) {
	exitStatusCode = n
}

// GetExitStatusCode waits for the shutdown to complete and then returns the exit code
func GetExitStatusCode() int {
	<-shutdownCompleteSignal
	return exitStatusCode
}
