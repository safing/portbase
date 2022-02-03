package info

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
)

var (
	name         = "[NAME]"
	version      = "[version unknown]"
	commit       = "[commit unknown]"
	license      = "[license unknown]"
	buildOptions = "[options unknown]"
	buildUser    = "[user unknown]"
	buildHost    = "[host unknown]"
	buildDate    = "[date unknown]"
	buildSource  = "[source unknown]"

	compareVersion bool
)

// Info holds the programs meta information.
type Info struct {
	Name         string
	Version      string
	License      string
	Commit       string
	BuildOptions string
	BuildUser    string
	BuildHost    string
	BuildDate    string
	BuildSource  string
}

// Set sets meta information via the main routine. This should be the first thing your program calls.
func Set(setName string, setVersion string, setLicenseName string, compareVersionToTag bool) {
	name = setName
	version = setVersion
	license = setLicenseName
	compareVersion = compareVersionToTag
}

// GetInfo returns all the meta information about the program.
func GetInfo() *Info {
	return &Info{
		Name:         name,
		Version:      version,
		Commit:       commit,
		License:      license,
		BuildOptions: buildOptions,
		BuildUser:    buildUser,
		BuildHost:    buildHost,
		BuildDate:    buildDate,
		BuildSource:  buildSource,
	}
}

// Version returns the short version string.
func Version() string {
	if !compareVersion || strings.HasPrefix(commit, fmt.Sprintf("tags/v%s-0-", version)) {
		return version
	}
	return version + "*"
}

// FullVersion returns the full and detailed version string.
func FullVersion() string {
	s := ""
	if !compareVersion || strings.HasPrefix(commit, fmt.Sprintf("tags/v%s-0-", version)) {
		s += fmt.Sprintf("%s\nversion %s\n", name, version)
	} else {
		s += fmt.Sprintf("%s\ndevelopment build, built on top version %s\n", name, version)
	}
	s += fmt.Sprintf("\ncommit %s\n", commit)
	s += fmt.Sprintf("built with %s (%s) %s/%s\n", runtime.Version(), runtime.Compiler, runtime.GOOS, runtime.GOARCH)
	s += fmt.Sprintf("  using options %s\n", strings.ReplaceAll(buildOptions, "§", " "))
	s += fmt.Sprintf("  by %s@%s\n", buildUser, buildHost)
	s += fmt.Sprintf("  on %s\n", buildDate)
	s += fmt.Sprintf("\nLicensed under the %s license.\nThe source code is available here: %s", license, buildSource)
	return s
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
