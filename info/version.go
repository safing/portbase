package info

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
)

var (
	name        string
	version     = "dev build"
	buildSource = "[source unknown]"
	buildTime   = "[build time unknown]"
	license     = "[license unknown]"

	info     *Info
	loadInfo sync.Once
)

// Info holds the programs meta information.
type Info struct {
	Name    string
	Version string
	License string

	Source    string
	BuildTime string

	Commit     string
	CommitTime string
	Dirty      bool

	debug.BuildInfo
}

// Set sets meta information via the main routine. This should be the first thing your program calls.
func Set(setName string, setVersion string, setLicenseName string) {
	name = setName
	license = setLicenseName

	if setVersion != "" {
		version = setVersion
	}
}

// GetInfo returns all the meta information about the program.
func GetInfo() *Info {
	loadInfo.Do(func() {
		buildInfo, _ := debug.ReadBuildInfo()
		buildSettings := make(map[string]string)
		for _, setting := range buildInfo.Settings {
			buildSettings[setting.Key] = setting.Value
		}

		info = &Info{
			Name:       name,
			Version:    version,
			License:    license,
			Source:     buildSource,
			BuildTime:  buildTime,
			Commit:     buildSettings["vcs.revision"],
			CommitTime: buildSettings["vcs.time"],
			Dirty:      buildSettings["vcs.modified"] == "true",
			BuildInfo:  *buildInfo,
		}

		if info.Commit == "" {
			info.Commit = "[commit unknown]"
		}
		if info.CommitTime == "" {
			info.CommitTime = "[commit time unknown]"
		}
	})

	return info
}

// Version returns the short version string.
func Version() string {
	info := GetInfo()

	if info.Dirty {
		return version + "*"
	}

	return version
}

// FullVersion returns the full and detailed version string.
func FullVersion() string {
	info := GetInfo()
	builder := new(strings.Builder)

	// Name and version.
	builder.WriteString(fmt.Sprintf("%s %s\n", info.Name, Version()))

	// Build info.
	builder.WriteString(fmt.Sprintf("\nbuilt with %s (%s) %s/%s\n", runtime.Version(), runtime.Compiler, runtime.GOOS, runtime.GOARCH))
	builder.WriteString(fmt.Sprintf("  at %s\n", info.BuildTime))

	// Commit info.
	builder.WriteString(fmt.Sprintf("\ncommit %s\n", info.Commit))
	builder.WriteString(fmt.Sprintf("  at %s\n", info.CommitTime))
	builder.WriteString(fmt.Sprintf("  from %s\n", info.Source))

	builder.WriteString(fmt.Sprintf("\nLicensed under the %s license.", license))

	return builder.String()
}

// CheckVersion checks if the metadata is ok.
func CheckVersion() error {
	switch {
	case strings.HasSuffix(os.Args[0], ".test"):
		return nil // testing on linux/darwin
	case strings.HasSuffix(os.Args[0], ".test.exe"):
		return nil // testing on windows
	default:
		// check version information
		if name == "[NAME]" || license == "[license unknown]" {
			return errors.New("must call SetInfo() before calling CheckVersion()")
		}
	}

	return nil
}
