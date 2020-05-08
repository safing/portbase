package dbmodule

import (
	"context"
	"time"

	"github.com/safing/portbase/database"
	"github.com/safing/portbase/log"
	"github.com/safing/portbase/modules"
)

func startMaintenanceTasks() {
	module.NewTask("basic maintenance", maintainBasic).Repeat(10 * time.Minute).MaxDelay(10 * time.Minute)
	module.NewTask("thorough maintenance", maintainThorough).Repeat(1 * time.Hour).MaxDelay(1 * time.Hour)
	module.NewTask("record maintenance", maintainRecords).Repeat(1 * time.Hour).MaxDelay(1 * time.Hour)
}

func maintainBasic(ctx context.Context, task *modules.Task) error {
	log.Infof("database: running Maintain")
	return database.Maintain(ctx)
}

func maintainThorough(ctx context.Context, task *modules.Task) error {
	log.Infof("database: running MaintainThorough")
	return database.MaintainThorough(ctx)
}

func maintainRecords(ctx context.Context, task *modules.Task) error {
	log.Infof("database: running MaintainRecordStates")
	return database.MaintainRecordStates(ctx)
}
