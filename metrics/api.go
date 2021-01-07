package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/safing/portbase/api"
	"github.com/safing/portbase/config"
	"github.com/safing/portbase/log"
)

func registerAPI() error {
	api.RegisterHandler("/metrics", &metricsAPI{})

	return api.RegisterEndpoint(api.Endpoint{
		Path:     "metrics/list",
		Read:     api.PermitAnyone,
		MimeType: api.MimeTypeJSON,
		DataFunc: func(*api.Request) ([]byte, error) {
			registryLock.RLock()
			defer registryLock.RUnlock()

			return json.Marshal(registry)
		},
		Name:        "Export Registered Metrics",
		Description: "List all registered metrics with their metadata.",
	})
}

type metricsAPI struct{}

func (m *metricsAPI) ReadPermission(*http.Request) api.Permission { return api.Dynamic }

func (m *metricsAPI) WritePermission(*http.Request) api.Permission { return api.NotSupported }

func (m *metricsAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get API Request for permission and query.
	ar := api.GetAPIRequest(r)
	if ar == nil {
		http.Error(w, "Missing API Request.", http.StatusInternalServerError)
		return
	}

	// Get expertise level from query.
	expertiseLevel := config.ExpertiseLevelDeveloper
	switch ar.Request.URL.Query().Get("level") {
	case config.ExpertiseLevelNameUser:
		expertiseLevel = config.ExpertiseLevelUser
	case config.ExpertiseLevelNameExpert:
		expertiseLevel = config.ExpertiseLevelExpert
	case config.ExpertiseLevelNameDeveloper:
		expertiseLevel = config.ExpertiseLevelDeveloper
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	WriteMetrics(w, ar.AuthToken.Read, expertiseLevel)
}

// WriteMetrics writes all metrics that match the given permission and
// expertiseLevel to the given writer.
func WriteMetrics(w io.Writer, permission api.Permission, expertiseLevel config.ExpertiseLevel) {
	registryLock.RLock()
	defer registryLock.RUnlock()

	// Check if metric ID is already registered.
	for _, metric := range registry {
		if permission >= metric.Opts().Permission &&
			expertiseLevel >= metric.Opts().ExpertiseLevel {
			metric.WritePrometheus(w)
		}
	}
}

func writeMetricsTo(ctx context.Context, url string) error {
	// First, collect metrics into buffer.
	buf := &bytes.Buffer{}
	WriteMetrics(buf, api.PermitSelf, config.ExpertiseLevelDeveloper)

	// Check if there is something to send.
	if buf.Len() == 0 {
		log.Debugf("metrics: not pushing metrics, nothing to send")
		return nil
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, buf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Send.
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check return status.
	switch resp.StatusCode {
	case http.StatusOK,
		http.StatusAccepted,
		http.StatusNoContent:
		return nil
	default:
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf(
			"got %s while writing metrics to %s: %s",
			resp.Status,
			url,
			body,
		)
	}
}

func metricsWriter(ctx context.Context) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			err := writeMetricsTo(ctx, pushURL)
			if err != nil {
				return err
			}
		}
	}
}
