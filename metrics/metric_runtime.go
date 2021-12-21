package metrics

import (
	"io"

	vm "github.com/VictoriaMetrics/metrics"

	"github.com/safing/portbase/api"
	"github.com/safing/portbase/config"
)

func registerRuntimeMetric() error {
	runtimeBase, err := newMetricBase("_runtime", nil, Options{
		Name:           "Golang Runtime",
		Permission:     api.PermitAdmin,
		ExpertiseLevel: config.ExpertiseLevelDeveloper,
	})
	if err != nil {
		return err
	}

	return register(&runtimeMetrics{
		metricBase: runtimeBase,
	})
}

type runtimeMetrics struct {
	*metricBase
}

func (r *runtimeMetrics) WritePrometheus(w io.Writer) {
	// TODO: Add global labels.
	vm.WriteProcessMetrics(w)
}
