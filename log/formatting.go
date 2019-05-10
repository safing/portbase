// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package log

import (
	"fmt"
	"time"
)

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
		fLine = fmt.Sprintf("%s%s ? %s %s %03d%s%s %s", colorStart, line.timestamp.Format("060102 15:04:05.000"), rightArrow, line.level.String(), counter, formatDuplicates(duplicates), colorEnd, line.msg)
	} else {
		fLen := len(line.file)
		fPartStart := fLen - 10
		if fPartStart < 0 {
			fPartStart = 0
		}
		fLine = fmt.Sprintf("%s%s %s:%03d %s %s %03d%s%s %s", colorStart, line.timestamp.Format("060102 15:04:05.000"), line.file[fPartStart:], line.line, rightArrow, line.level.String(), counter, formatDuplicates(duplicates), colorEnd, line.msg)
	}

	if line.trace != nil {
		// append full trace time
		if len(line.trace.actions) > 0 {
			fLine += fmt.Sprintf(" Σ=%s", line.timestamp.Sub(line.trace.actions[0].timestamp))
		}

		// append all trace actions
		var d time.Duration
		for i, action := range line.trace.actions {
			// set color
			if useColor {
				colorStart = action.level.color()
			}
			// set filename length
			fLen := len(action.file)
			fPartStart := fLen - 10
			if fPartStart < 0 {
				fPartStart = 0
			}
			// format
			if i == len(line.trace.actions)-1 { // last
				d = line.timestamp.Sub(action.timestamp)
			} else {
				d = line.trace.actions[i+1].timestamp.Sub(action.timestamp)
			}
			fLine += fmt.Sprintf("\n%s%19s %s:%03d %s %s%s     %s", colorStart, d, action.file[fPartStart:], action.line, rightArrow, action.level.String(), colorEnd, action.msg)
		}
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
