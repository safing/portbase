package dbmodule

import (
  "time"

  "github.com/Safing/portbase/database"
  "github.com/Safing/portbase/log"
)

func maintainer() {
  ticker := time.NewTicker(1 * time.Hour)
  tickerThorough := time.NewTicker(10 * time.Minute)
  maintenanceWg.Add(1)

  for {
    select {
    case <- ticker.C:
      err := database.Maintain()
      if err != nil {
        log.Errorf("database: maintenance error: %s", err)
      }
    case <- ticker.C:
      err := database.MaintainThorough()
      if err != nil {
        log.Errorf("database: maintenance (thorough) error: %s", err)
      }
    case <-shutdownSignal:
      maintenanceWg.Done()
      return
    }
  }
}
