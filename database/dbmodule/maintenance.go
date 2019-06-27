package dbmodule

import (
	"time"

	"github.com/safing/portbase/database"
	"github.com/safing/portbase/log"
)

var (
	maintenanceShortTickDuration = 10 * time.Minute
	maintenanceLongTickDuration  = 1 * time.Hour
)

func startMaintainer() {
	maintenanceWg.Add(1)
	go maintenanceWorker()
}

func maintenanceWorker() {
	ticker := time.NewTicker(maintenanceShortTickDuration)
	longTicker := time.NewTicker(maintenanceLongTickDuration)

	for {
		select {
		case <-ticker.C:
			err := database.Maintain()
			if err != nil {
				log.Errorf("database: maintenance error: %s", err)
			}
		case <-longTicker.C:
			err := database.MaintainRecordStates()
			if err != nil {
				log.Errorf("database: record states maintenance error: %s", err)
			}
			err = database.MaintainThorough()
			if err != nil {
				log.Errorf("database: thorough maintenance error: %s", err)
			}
		case <-shutdownSignal:
			maintenanceWg.Done()
			return
		}
	}
}
