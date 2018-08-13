// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package log

var Logger *LoggingInterface

type LoggingInterface struct {
}

func (*LoggingInterface) Tracef(things ...interface{}) {
	Tracef(things...)
}

func (*LoggingInterface) Trace(msg string) {
	Trace(msg)
}

func (*LoggingInterface) Debugf(things ...interface{}) {
	Debugf(things...)
}

func (*LoggingInterface) Debug(msg string) {
	Debug(msg)
}

func (*LoggingInterface) Infof(things ...interface{}) {
	Infof(things...)
}

func (*LoggingInterface) Info(msg string) {
	Info(msg)
}

func (*LoggingInterface) Warningf(things ...interface{}) {
	Warningf(things...)
}

func (*LoggingInterface) Warning(msg string) {
	Warning(msg)
}

func (*LoggingInterface) Errorf(things ...interface{}) {
	Errorf(things...)
}

func (*LoggingInterface) Error(msg string) {
	Error(msg)
}

func (*LoggingInterface) Criticalf(things ...interface{}) {
	Criticalf(things...)
}

func (*LoggingInterface) Critical(msg string) {
	Critical(msg)
}

func init() {
	Logger = &LoggingInterface{}
}
