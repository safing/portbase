//go:build !windows

package osdetail

// GetBinaryNameFromSystem queries the operating system for a human readable
// name for the given binary path.
func GetBinaryNameFromSystem(path string) (string, error) {
	return "", ErrNotSupported
}

// GetBinaryIconFromSystem queries the operating system for the associated icon
// for a given binary path.
func GetBinaryIconFromSystem(path string) (string, error) {
	return "", ErrNotSupported
}
