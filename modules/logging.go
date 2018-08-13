// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package modules

var logger Logger
var loggerRegistered chan struct{}

type Logger interface {
	Tracef(things ...interface{})
	Trace(msg string)
	Debugf(things ...interface{})
	Debug(msg string)
	Infof(things ...interface{})
	Info(msg string)
	Warningf(things ...interface{})
	Warning(msg string)
	Errorf(things ...interface{})
	Error(msg string)
	Criticalf(things ...interface{})
	Critical(msg string)
}

func RegisterLogger(newLogger Logger) {
	if logger == nil {
		logger = newLogger
		loggerRegistered <- struct{}{}
	}
}

func GetLogger() Logger {
	return logger
}
