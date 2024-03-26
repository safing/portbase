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
	license     = "[license unknown]"

	info     *Info
	loadInfo sync.Once
)

// Info holds the programs meta information.
type Info struct {
	Name    string
	Version string
	License string
	Commit  string
	Time    string
	Source  string
	Dirty   bool

	debug.BuildInfo
}

// Set sets meta information via the main routine. This should be the first thing your program calls.
func Set(setName string, setVersion string, setLicenseName string, compareVersionToTag bool) {
	name = setName
	version = setVersion
	license = setLicenseName
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
			Name:      name,
			Version:   version,
			License:   license,
			BuildInfo: *buildInfo,
			Source:    buildSource,
			Commit:    buildSettings["vcs.revision"],
			Time:      buildSettings["vcs.time"],
			Dirty:     buildSettings["vcs.modified"] == "true",
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

	builder.WriteString(fmt.Sprintf("%s\nversion %s\n", info.Name, Version()))
	builder.WriteString(fmt.Sprintf("\ncommit %s\n", info.Commit))
	builder.WriteString(fmt.Sprintf("built with %s (%s) %s/%s\n", runtime.Version(), runtime.Compiler, runtime.GOOS, runtime.GOARCH))
	builder.WriteString(fmt.Sprintf("  on %s\n", info.Time))
	builder.WriteString(fmt.Sprintf("\nLicensed under the %s license.\nThe source code is available here: %s", license, info.Source))

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
		if name == "[NAME]" {
			return errors.New("must call SetInfo() before calling CheckVersion()")
		}

		if version == "[version unknown]" ||
			license == "[license unknown]" {
			return errors.New("please build using the supplied build script.\n$ ./build {main.go|...}")
		}
	}

	return nil
}
