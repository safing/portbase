// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package log

type color int

const (
	// colorBlack   = "\033[30m"
	colorRed = "\033[31m"
	// colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
	// colorWhite   = "\033[37m"
)

func (s severity) color() string {
	switch s {
	case DebugLevel:
		return colorCyan
	case InfoLevel:
		return colorBlue
	case WarningLevel:
		return colorYellow
	case ErrorLevel:
		return colorRed
	case CriticalLevel:
		return colorMagenta
	default:
		return ""
	}
}

func endColor() string {
	return "\033[0m"
}
