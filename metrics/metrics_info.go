package metrics

import (
	"runtime"
	"strings"

	"github.com/safing/portbase/info"
)

func registerInfoMetric() error {
	meta := info.GetInfo()
	_, err := NewGauge(
		"info",
		map[string]string{
			"version":       checkUnknown(meta.Version),
			"commit":        checkUnknown(meta.Commit),
			"build_options": checkUnknown(meta.BuildOptions),
			"build_user":    checkUnknown(meta.BuildUser),
			"build_host":    checkUnknown(meta.BuildHost),
			"build_date":    checkUnknown(meta.BuildDate),
			"build_source":  checkUnknown(meta.BuildSource),
			"go_os":         runtime.GOOS,
			"go_arch":       runtime.GOARCH,
			"go_version":    runtime.Version(),
			"go_compiler":   runtime.Compiler,
			"comment":       commentOption(),
		},
		func() float64 {
			return 1
		},
		nil,
	)
	return err
}

func checkUnknown(s string) string {
	if strings.Contains(s, "unknown") {
		return "unknown"
	}
	return s
}
