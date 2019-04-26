package osdetail

import (
	"os/exec"
	"regexp"
	"strings"
	"sync"
)

var (
	versionRe = regexp.MustCompile(`[0-9\.]+`)

	windowsVersion string

	fetching sync.Mutex
	fetched  bool
)

func fetchVersion() {
	if !fetched {
		fetched = true

		output, err := exec.Command("cmd", "ver").Output()
		if err != nil {
			return
		}

		match := versionRe.Find(output)
		if match == nil {
			return
		}

		windowsVersion = string(match)
	}
}

// WindowsVersion returns the current Windows version.
func WindowsVersion() string {
	fetching.Lock()
	defer fetching.Unlock()
	fetchVersion()

	return windowsVersion
}

// IsWindowsVersion returns whether the given version matches (HasPrefix) the current Windows version.
func IsWindowsVersion(version string) bool {
	fetching.Lock()
	defer fetching.Unlock()
	fetchVersion()

	// TODO: we can do better.
	return strings.HasPrefix(windowsVersion, version)
}
