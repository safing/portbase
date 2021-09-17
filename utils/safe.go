package utils

import (
	"encoding/hex"
	"strings"
)

func SafeFirst16Bytes(data []byte) string {
	if len(data) == 0 {
		return "<empty>"
	}

	return strings.TrimPrefix(
		strings.SplitN(hex.Dump(data), "\n", 2)[0],
		"00000000  ",
	)
}

func SafeFirst16Chars(s string) string {
	return SafeFirst16Bytes([]byte(s))
}
