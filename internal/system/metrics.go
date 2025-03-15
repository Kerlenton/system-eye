package system

import (
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
)

// GetCPUUsage returns the CPU usage percentage over a 1-second interval.
func GetCPUUsage() (float64, error) {
	percentages, err := cpu.Percent(time.Second, false)
	if err != nil {
		return 0, err
	}
	if len(percentages) == 0 {
		return 0, nil
	}
	return percentages[0], nil
}

// GetMemoryUsage returns the percentage of used memory.
func GetMemoryUsage() (float64, error) {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return 0, err
	}
	return vmStat.UsedPercent, nil
}

// GetDiskUsage returns the percentage of disk usage for the specified path.
func GetDiskUsage(path string) (float64, error) {
	usageStat, err := disk.Usage(path)
	if err != nil {
		return 0, err
	}
	return usageStat.UsedPercent, nil
}
