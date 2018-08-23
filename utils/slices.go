package utils

// StringInSlice returns whether the given string is in the string slice.
func StringInSlice(a []string, s string) bool {
	for _, entry := range a {
		if entry == s {
			return true
		}
	}
	return false
}

// RemoveFromStringSlice removes the given string from the slice and returns a new slice.
func RemoveFromStringSlice(a []string, s string) []string {
	for key, entry := range a {
		if entry == s {
			a = append(a[:key], a[key+1:]...)
			return a
		}
	}
	return a
}

// DuplicateStrings returns a new copy of the given string slice.
func DuplicateStrings(a []string) []string {
	b := make([]string, len(a))
	copy(b, a)
	return b
}

// StringSliceEqual returns whether the given string slices are equal.
func StringSliceEqual(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// DuplicateBytes returns a new copy of the given byte slice.
func DuplicateBytes(a []byte) []byte {
	b := make([]byte, len(a))
	copy(b, a)
	return b
}
