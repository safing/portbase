package dbmodule

import (
  "time"

  "github.com/Safing/portbase/database"
  "github.com/Safing/portbase/log"
)

func maintainer() {
  ticker := time.NewTicker(10 * time.Minute)
  longTicker := time.NewTicker(1 * time.Hour)
  maintenanceWg.Add(1)

  for {
    select {
    case <- ticker.C:
      err := database.Maintain()
      if err != nil {
        log.Errorf("database: maintenance error: %s", err)
      }
    case <- longTicker.C:
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
