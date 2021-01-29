package metrics

import (
	"io"

	vm "github.com/VictoriaMetrics/metrics"
	"github.com/safing/portbase/api"
	"github.com/safing/portbase/config"
)

func init() {
	registryLock.Lock()
	defer registryLock.Unlock()

	registry = append(registry, &runtimeMetrics{})
}

var runtimeOpts = &Options{
	Name:           "Golang Runtime",
	Permission:     api.PermitAdmin,
	ExpertiseLevel: config.ExpertiseLevelDeveloper,
}

type runtimeMetrics struct{}

func (r *runtimeMetrics) ID() string {
	return "_runtime"
}

func (r *runtimeMetrics) LabeledID() string {
	return "_runtime"
}

func (r *runtimeMetrics) Opts() *Options {
	return runtimeOpts
}

func (r *runtimeMetrics) Permission() api.Permission {
	return runtimeOpts.Permission
}

func (r *runtimeMetrics) ExpertiseLevel() config.ExpertiseLevel {
	return runtimeOpts.ExpertiseLevel
}

func (r *runtimeMetrics) WritePrometheus(w io.Writer) {
	vm.WriteProcessMetrics(w)
}
