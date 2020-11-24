// Package modules provides a full module and task management ecosystem to
// cleanly put all big and small moving parts of a service together.
//
// Modules are started in a multi-stage process and may depend on other
// modules:
// - Go's init(): register flags
// - prep: check flags, register config variables
// - start: start actual work, access config
// - stop: gracefully shut down
//
// **Workers**
// A simple function that is run by the module while catching
// panics and reporting them. Ideal for long running (possibly) idle goroutines.
// Can be automatically restarted if execution ends with an error.
//
// **Tasks**
// Functions that take somewhere between a couple seconds and a couple
// minutes to execute and should be queued, scheduled or repeated.
//
// **MicroTasks**
// Functions that take less than a second to execute, but require
// lots of resources. Running such functions as MicroTasks will reduce concurrent
// execution and shall improve performance.
//
// Ideally, _any_ execution by a module is done through these methods. This will
// not only ensure that all panics are caught, but will also give better insights
// into how your service performs.
package modules
