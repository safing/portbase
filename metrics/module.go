package metrics

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/safing/portbase/modules"
)

var (
	module *modules.Module

	registry     []Metric
	registryLock sync.RWMutex

	firstMetricRegistered bool
	metricNamespace       string
	globalLabels          = make(map[string]string)

	// ErrAlreadyStarted is returned when an operation is only valid before the
	// first metric is registered, and is called after.
	ErrAlreadyStarted = errors.New("can only be changed before first metric is registered")

	// ErrAlreadyRegistered is returned when a metric with the same ID is
	// registered again.
	ErrAlreadyRegistered = errors.New("metric already registered")

	// ErrAlreadySet is returned when a value is already set and cannot be changed.
	ErrAlreadySet = errors.New("already set")

	// ErrInvalidOptions is returned when invalid options where provided.
	ErrInvalidOptions = errors.New("invalid options")
)

func init() {
	module = modules.Register("metrics", prep, start, stop, "config", "database", "api")
}

func prep() error {
	return prepConfig()
}

func start() error {
	// Add metric instance name as global variable if set.
	if instanceOption() != "" {
		if err := AddGlobalLabel("instance", instanceOption()); err != nil {
			return err
		}
	}

	if err := registerInfoMetric(); err != nil {
		return err
	}

	if err := registerRuntimeMetric(); err != nil {
		return err
	}

	if err := registeHostMetrics(); err != nil {
		return err
	}

	if err := registeLogMetrics(); err != nil {
		return err
	}

	if err := registerAPI(); err != nil {
		return err
	}

	if pushOption() != "" {
		module.StartServiceWorker("metric pusher", 0, metricsWriter)
	}

	return nil
}

func stop() error {
	// Wait until the metrics pusher is done, as it may have started reporting
	// and may report a higher number than we store to disk. For persistent
	// metrics it can then happen that the first report is lower than the
	// previous report, making prometheus think that al that happened since the
	// last report, due to the automatic restart detection.
	done := metricsPusherDone.NewFlag()
	done.Refresh()
	if !done.IsSet() {
		select {
		case <-done.Signal():
		case <-time.After(10 * time.Second):
		}
	}

	storePersistentMetrics()

	return nil
}

func register(m Metric) error {
	registryLock.Lock()
	defer registryLock.Unlock()

	// Check if metric ID is already registered.
	for _, registeredMetric := range registry {
		if m.LabeledID() == registeredMetric.LabeledID() {
			return ErrAlreadyRegistered
		}
		if m.Opts().InternalID != "" &&
			m.Opts().InternalID == registeredMetric.Opts().InternalID {
			return fmt.Errorf("%w with this internal ID", ErrAlreadyRegistered)
		}
	}

	// Add new metric to registry and sort it.
	registry = append(registry, m)
	sort.Sort(byLabeledID(registry))

	// Set flag that first metric is now registered.
	firstMetricRegistered = true

	return nil
}

// SetNamespace sets the namespace for all metrics. It is prefixed to all
// metric IDs.
// It must be set before any metric is registered.
// Does not affect golang runtime metrics.
func SetNamespace(namespace string) error {
	// Lock registry and check if a first metric is already registered.
	registryLock.Lock()
	defer registryLock.Unlock()
	if firstMetricRegistered {
		return ErrAlreadyStarted
	}

	// Check if the namespace is already set.
	if metricNamespace != "" {
		return ErrAlreadySet
	}

	metricNamespace = namespace
	return nil
}

// AddGlobalLabel adds a global label to all metrics.
// Global labels must be added before any metric is registered.
// Does not affect golang runtime metrics.
func AddGlobalLabel(name, value string) error {
	// Lock registry and check if a first metric is already registered.
	registryLock.Lock()
	defer registryLock.Unlock()
	if firstMetricRegistered {
		return ErrAlreadyStarted
	}

	// Check format.
	if !prometheusFormat.MatchString(name) {
		return fmt.Errorf("metric label name %q must match %s", name, PrometheusFormatRequirement)
	}

	globalLabels[name] = value
	return nil
}

type byLabeledID []Metric

func (r byLabeledID) Len() int           { return len(r) }
func (r byLabeledID) Less(i, j int) bool { return r[i].LabeledID() < r[j].LabeledID() }
func (r byLabeledID) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }
