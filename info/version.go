package info

import (
	"fmt"
	"runtime"
	"strings"
)

var (
	name         = "[NAME]"
	version      = "[version unknown]"
	commit       = "[commit unknown]"
	buildOptions = "[options unknown]"
	buildUser    = "[user unknown]"
	buildHost    = "[host unknown]"
	buildDate    = "[date unknown]"
	buildSource  = "[source unknown]"
)

type Info struct {
	Name         string
	Version      string
	Commit       string
	BuildOptions string
	BuildUser    string
	BuildHost    string
	BuildDate    string
	BuildSource  string
}

func Set(setName string, setVersion string) {
	name = setName
	version = setVersion
}

func GetInfo() *Info {
	return &Info{
		Name:         name,
		Version:      version,
		Commit:       commit,
		BuildOptions: buildOptions,
		BuildUser:    buildUser,
		BuildHost:    buildHost,
		BuildDate:    buildDate,
		BuildSource:  buildSource,
	}
}

func Version() string {
	if strings.HasPrefix(commit, fmt.Sprintf("v%s-0-", version)) {
		return version
	} else {
		return version + "*"
	}
}

func FullVersion() string {
	s := ""
	if strings.HasPrefix(commit, fmt.Sprintf("v%s-0-", version)) {
		s += fmt.Sprintf("%s\nversion %s\n", name, version)
	} else {
		s += fmt.Sprintf("%s\ndevelopment build, built on top version %s\n", name, version)
	}
	s += fmt.Sprintf("\ncommit %s\n", commit)
	s += fmt.Sprintf("built with %s (%s) %s/%s\n", runtime.Version(), runtime.Compiler, runtime.GOOS, runtime.GOARCH)
	s += fmt.Sprintf("  using options %s\n", strings.Replace(buildOptions, "§", " ", -1))
	s += fmt.Sprintf("  by %s@%s\n", buildUser, buildHost)
	s += fmt.Sprintf("  on %s\n", buildDate)
	s += fmt.Sprintf("\nLicensed under the AGPL license.\nThe source code is available here: %s", buildSource)
	return s
}
