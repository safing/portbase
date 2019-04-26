package osdetail

import "testing"

func TestWindowsVersion(t *testing.T) {
	if WindowsVersion() == "" {
		t.Fatal("could not get windows version")
	}
}
