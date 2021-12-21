package metrics

import (
	"github.com/safing/portbase/api"
	"github.com/safing/portbase/log"
)

func registeLogMetrics() (err error) {
	_, err = NewFetchingCounter(
		"logs/warning/total",
		nil,
		log.TotalWarningLogLines,
		&Options{
			Name:       "Total Warning Log Lines",
			Permission: api.PermitUser,
		},
	)
	if err != nil {
		return err
	}

	_, err = NewFetchingCounter(
		"logs/error/total",
		nil,
		log.TotalErrorLogLines,
		&Options{
			Name:       "Total Error Log Lines",
			Permission: api.PermitUser,
		},
	)
	if err != nil {
		return err
	}

	_, err = NewFetchingCounter(
		"logs/critical/total",
		nil,
		log.TotalCriticalLogLines,
		&Options{
			Name:       "Total Critical Log Lines",
			Permission: api.PermitUser,
		},
	)
	if err != nil {
		return err
	}

	return nil
}
