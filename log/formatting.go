// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package log

import "fmt"

var counter uint16

const maxCount uint16 = 999

func (s severity) String() string {
	switch s {
	case TraceLevel:
		return "TRAC"
	case DebugLevel:
		return "DEBU"
	case InfoLevel:
		return "INFO"
	case WarningLevel:
		return "WARN"
	case ErrorLevel:
		return "ERRO"
	case CriticalLevel:
		return "CRIT"
	default:
		return "NONE"
	}
}

func formatLine(line *logLine, duplicates uint64, useColor bool) string {

	colorStart := ""
	colorEnd := ""
	if useColor {
		colorStart = line.level.color()
		colorEnd = endColor()
	}

	counter++

	var fLine string
	if line.line == 0 {
		fLine = fmt.Sprintf("%s%s ? %s %s %03d%s%s %s", colorStart, line.time.Format("060102 15:04:05.000"), rightArrow, line.level.String(), counter, formatDuplicates(duplicates), colorEnd, line.msg)
	} else {
		fLen := len(line.file)
		fPartStart := fLen - 10
		if fPartStart < 0 {
			fPartStart = 0
		}
		fLine = fmt.Sprintf("%s%s %s:%03d %s %s %03d%s%s %s", colorStart, line.time.Format("060102 15:04:05.000"), line.file[fPartStart:], line.line, rightArrow, line.level.String(), counter, formatDuplicates(duplicates), colorEnd, line.msg)
	}

	if counter >= maxCount {
		counter = 0
	}

	return fLine
}

func formatDuplicates(duplicates uint64) string {
	if duplicates == 0 {
		return ""
	}
	return fmt.Sprintf(" [%dx]", duplicates+1)
}
