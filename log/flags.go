package log

import "flag"

var (
	logLevelFlag      string
	fileLogLevelsFlag string
)

func init() {
	flag.StringVar(&logLevelFlag, "log", "info", "set log level to [trace|debug|info|warning|error|critical]")
	flag.StringVar(&fileLogLevelsFlag, "flog", "", "set log level of files: database=trace,firewall=debug")
}
