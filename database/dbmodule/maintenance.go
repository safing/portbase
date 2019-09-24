package dbmodule

import (
	"context"
	"time"

	"github.com/safing/portbase/database"
	"github.com/safing/portbase/log"
	"github.com/safing/portbase/modules"
)

func registerMaintenanceTasks() {
	module.NewTask("basic maintenance", maintainBasic).Repeat(10 * time.Minute).MaxDelay(10 * time.Minute)
	module.NewTask("thorough maintenance", maintainThorough).Repeat(1 * time.Hour).MaxDelay(1 * time.Hour)
	module.NewTask("record maintenance", maintainRecords).Repeat(1 * time.Hour).MaxDelay(1 * time.Hour)
}

func maintainBasic(ctx context.Context, task *modules.Task) {
	err := database.Maintain()
	if err != nil {
		log.Errorf("database: maintenance error: %s", err)
	}
}

func maintainThorough(ctx context.Context, task *modules.Task) {
	err := database.MaintainThorough()
	if err != nil {
		log.Errorf("database: thorough maintenance error: %s", err)
	}
}

func maintainRecords(ctx context.Context, task *modules.Task) {
	err := database.MaintainRecordStates()
	if err != nil {
		log.Errorf("database: record states maintenance error: %s", err)
	}
}
