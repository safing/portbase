package api

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"runtime/pprof"

	"github.com/safing/portbase/utils/debug"
)

func registerDebugEndpoints() error {
	if err := RegisterEndpoint(Endpoint{
		Path:        "debug/stack",
		Read:        PermitAnyone,
		DataFunc:    getStack,
		Name:        "Get Goroutine Stack",
		Description: "Returns the current goroutine stack.",
	}); err != nil {
		return err
	}

	if err := RegisterEndpoint(Endpoint{
		Path:        "debug/stack/print",
		Read:        PermitAnyone,
		ActionFunc:  printStack,
		Name:        "Print Goroutine Stack",
		Description: "Prints the current goroutine stack to stdout.",
	}); err != nil {
		return err
	}

	if err := RegisterEndpoint(Endpoint{
		Path:        "debug/info",
		Read:        PermitAnyone,
		DataFunc:    debugInfo,
		Name:        "Get Debug Information",
		Description: "Returns debugging information, including the version and platform info, errors, logs and the current goroutine stack.",
		Parameters: []Parameter{{
			Method:      http.MethodGet,
			Field:       "style",
			Value:       "github",
			Description: "Specify the formatting style. The default is simple markdown formatting.",
		}},
	}); err != nil {
		return err
	}

	return nil
}

// getStack returns the current goroutine stack.
func getStack(_ *Request) (data []byte, err error) {
	buf := &bytes.Buffer{}
	err = pprof.Lookup("goroutine").WriteTo(buf, 1)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// printStack prints the current goroutine stack to stderr.
func printStack(_ *Request) (msg string, err error) {
	_, err = fmt.Fprint(os.Stderr, "===== PRINTING STACK =====\n")
	if err == nil {
		err = pprof.Lookup("goroutine").WriteTo(os.Stderr, 1)
	}
	if err == nil {
		_, err = fmt.Fprint(os.Stderr, "===== END OF STACK =====\n")
	}
	if err != nil {
		return "", err
	}
	return "stack printed to stdout", nil
}

// debugInfo returns the debugging information for support requests.
func debugInfo(ar *Request) (data []byte, err error) {
	// Create debug information helper.
	di := new(debug.Info)
	di.Style = ar.Request.URL.Query().Get("style")

	// Add debug information.
	di.AddVersionInfo()
	di.AddPlatformInfo(ar.Context())
	di.AddLastReportedModuleError()
	di.AddLastUnexpectedLogs()
	di.AddGoroutineStack()

	// Return data.
	return di.Bytes(), nil
}
