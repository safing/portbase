package testutils

import "runtime"

func GetLineNumberOfCaller(levels int) int {
	_, _, line, _ := runtime.Caller(levels + 1)
	return line
}
