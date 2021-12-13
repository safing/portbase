package metrics

import (
	"runtime"
	"sync"
	"time"

	"github.com/safing/portbase/api"
	"github.com/safing/portbase/dataroot"

	"github.com/safing/portbase/log"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
)

const hostStatTTL = 1 * time.Second

func registeHostMetrics() (err error) {
	// Register load average metrics.
	_, err = NewGauge("host_load_avg_1", nil, getFloat64HostStat(LoadAvg1), &Options{Name: "Host Load Avg 1min", Permission: api.PermitUser})
	if err != nil {
		return err
	}
	_, err = NewGauge("host_load_avg_5", nil, getFloat64HostStat(LoadAvg5), &Options{Name: "Host Load Avg 5min", Permission: api.PermitUser})
	if err != nil {
		return err
	}
	_, err = NewGauge("host_load_avg_15", nil, getFloat64HostStat(LoadAvg15), &Options{Name: "Host Load Avg 15min", Permission: api.PermitUser})
	if err != nil {
		return err
	}

	// Register memory usage metrics.
	_, err = NewGauge("host_mem_total", nil, getUint64HostStat(MemTotal), &Options{Name: "Host Memory Total", Permission: api.PermitUser})
	if err != nil {
		return err
	}
	_, err = NewGauge("host_mem_used", nil, getUint64HostStat(MemUsed), &Options{Name: "Host Memory Used", Permission: api.PermitUser})
	if err != nil {
		return err
	}
	_, err = NewGauge("host_mem_available", nil, getUint64HostStat(MemAvailable), &Options{Name: "Host Memory Available", Permission: api.PermitUser})
	if err != nil {
		return err
	}
	_, err = NewGauge("host_mem_used_percent", nil, getFloat64HostStat(MemUsedPercent), &Options{Name: "Host Memory Used in Percent", Permission: api.PermitUser})
	if err != nil {
		return err
	}

	// Register disk usage metrics.
	_, err = NewGauge("host_disk_total", nil, getUint64HostStat(DiskTotal), &Options{Name: "Host Disk Total", Permission: api.PermitUser})
	if err != nil {
		return err
	}
	_, err = NewGauge("host_disk_used", nil, getUint64HostStat(DiskUsed), &Options{Name: "Host Disk Used", Permission: api.PermitUser})
	if err != nil {
		return err
	}
	_, err = NewGauge("host_disk_free", nil, getUint64HostStat(DiskFree), &Options{Name: "Host Disk Free", Permission: api.PermitUser})
	if err != nil {
		return err
	}
	_, err = NewGauge("host_disk_used_percent", nil, getFloat64HostStat(DiskUsedPercent), &Options{Name: "Host Disk Used in Percent", Permission: api.PermitUser})
	if err != nil {
		return err
	}

	return nil
}

func getUint64HostStat(getStat func() (uint64, bool)) func() float64 {
	return func() float64 {
		val, _ := getStat()
		return float64(val)
	}
}

func getFloat64HostStat(getStat func() (float64, bool)) func() float64 {
	return func() float64 {
		val, _ := getStat()
		return val
	}
}

var (
	loadAvg        *load.AvgStat
	loadAvgExpires time.Time
	loadAvgLock    sync.Mutex
)

func getLoadAvg() *load.AvgStat {
	loadAvgLock.Lock()
	defer loadAvgLock.Unlock()

	// Return cache if still valid.
	if time.Now().Before(loadAvgExpires) {
		return loadAvg
	}

	// Refresh.
	var err error
	loadAvg, err = load.Avg()
	if err != nil {
		log.Warningf("metrics: failed to get load avg: %s", err)
		loadAvg = nil
	}
	loadAvgExpires = time.Now().Add(hostStatTTL)

	return loadAvg
}

func LoadAvg1() (loadAvg float64, ok bool) {
	if stat := getLoadAvg(); stat != nil {
		return stat.Load1 / float64(runtime.NumCPU()), true
	}
	return 0, false
}

func LoadAvg5() (loadAvg float64, ok bool) {
	if stat := getLoadAvg(); stat != nil {
		return stat.Load5 / float64(runtime.NumCPU()), true
	}
	return 0, false
}

func LoadAvg15() (loadAvg float64, ok bool) {
	if stat := getLoadAvg(); stat != nil {
		return stat.Load15 / float64(runtime.NumCPU()), true
	}
	return 0, false
}

var (
	memStat        *mem.VirtualMemoryStat
	memStatExpires time.Time
	memStatLock    sync.Mutex
)

func getMemStat() *mem.VirtualMemoryStat {
	memStatLock.Lock()
	defer memStatLock.Unlock()

	// Return cache if still valid.
	if time.Now().Before(memStatExpires) {
		return memStat
	}

	// Refresh.
	var err error
	memStat, err = mem.VirtualMemory()
	if err != nil {
		log.Warningf("metrics: failed to get load avg: %s", err)
		memStat = nil
	}
	memStatExpires = time.Now().Add(hostStatTTL)

	return memStat
}

func MemTotal() (total uint64, ok bool) {
	if stat := getMemStat(); stat != nil {
		return stat.Total, true
	}
	return 0, false
}

func MemUsed() (used uint64, ok bool) {
	if stat := getMemStat(); stat != nil {
		return stat.Used, true
	}
	return 0, false
}

func MemAvailable() (available uint64, ok bool) {
	if stat := getMemStat(); stat != nil {
		return stat.Available, true
	}
	return 0, false
}

func MemUsedPercent() (usedPercent float64, ok bool) {
	if stat := getMemStat(); stat != nil {
		return stat.UsedPercent, true
	}
	return 0, false
}

var (
	diskStat        *disk.UsageStat
	diskStatExpires time.Time
	diskStatLock    sync.Mutex
)

func getDiskStat() *disk.UsageStat {
	diskStatLock.Lock()
	defer diskStatLock.Unlock()

	// Return cache if still valid.
	if time.Now().Before(diskStatExpires) {
		return diskStat
	}

	// Check if we have a data root.
	dataRoot := dataroot.Root()
	if dataRoot == nil {
		log.Warning("metrics: cannot get disk stats without data root")
		diskStat = nil
		diskStatExpires = time.Now().Add(hostStatTTL)
		return diskStat
	}

	// Refresh.
	var err error
	diskStat, err = disk.Usage(dataRoot.Path)
	if err != nil {
		log.Warningf("metrics: failed to get load avg: %s", err)
		diskStat = nil
	}
	diskStatExpires = time.Now().Add(hostStatTTL)

	return diskStat
}

func DiskTotal() (total uint64, ok bool) {
	if stat := getDiskStat(); stat != nil {
		return stat.Total, true
	}
	return 0, false
}

func DiskUsed() (used uint64, ok bool) {
	if stat := getDiskStat(); stat != nil {
		return stat.Used, true
	}
	return 0, false
}

func DiskFree() (free uint64, ok bool) {
	if stat := getDiskStat(); stat != nil {
		return stat.Free, true
	}
	return 0, false
}

func DiskUsedPercent() (usedPercent float64, ok bool) {
	if stat := getDiskStat(); stat != nil {
		return stat.UsedPercent, true
	}
	return 0, false
}
