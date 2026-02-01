package monitor

import (
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	
	"neoagent/internal/pkg/logger"
)

// SystemMetrics 系统指标
type SystemMetrics struct {
	CPUUsage         float64
	MemoryUsage      float64
	DiskUsage        float64
	NetworkBytesSent int64
	NetworkBytesRecv int64
}

// GetSystemMetrics 获取系统指标
func GetSystemMetrics() (*SystemMetrics, error) {
	metrics := &SystemMetrics{}

	// 1. CPU Usage
	// Interval 0 means return immediately (might be inaccurate for the first call, but okay for heartbeat)
	// Better to use a small interval or keep state, but for stateless call:
	// To get accurate CPU usage, we typically need to sample over a duration.
	// Since this is a heartbeat, we can afford a small delay, or we can use the last value.
	// For simplicity, let's use 100ms sample.
	cpuPercent, err := cpu.Percent(100*time.Millisecond, false)
	if err != nil {
		logger.LogSystemEvent("Monitor", "GetSystemMetrics", "Failed to get CPU usage: "+err.Error(), logger.WarnLevel, nil)
	} else if len(cpuPercent) > 0 {
		metrics.CPUUsage = cpuPercent[0]
	}

	// 2. Memory Usage
	vMem, err := mem.VirtualMemory()
	if err != nil {
		logger.LogSystemEvent("Monitor", "GetSystemMetrics", "Failed to get Memory usage: "+err.Error(), logger.WarnLevel, nil)
	} else {
		metrics.MemoryUsage = vMem.UsedPercent
	}

	// 3. Disk Usage
	// We check the root path "/" (Linux) or "C:" (Windows)
	// For a cross-platform agent, we might need to be smarter, but root is a good proxy.
	diskPath := "/"
	// In Windows, gopsutil handles "/" as current drive or system drive usually, but let's be safe.
	// Actually gopsutil/disk.Usage("/") works on Windows too (it maps to C:\ or current drive).
	dUsage, err := disk.Usage(diskPath)
	if err != nil {
		// Fallback for Windows if "/" fails
		dUsage, err = disk.Usage("C:")
	}
	
	if err != nil {
		logger.LogSystemEvent("Monitor", "GetSystemMetrics", "Failed to get Disk usage: "+err.Error(), logger.WarnLevel, nil)
	} else {
		metrics.DiskUsage = dUsage.UsedPercent
	}

	// 4. Network Stats
	// We get counters for all interfaces combined
	netIO, err := net.IOCounters(false)
	if err != nil {
		logger.LogSystemEvent("Monitor", "GetSystemMetrics", "Failed to get Network stats: "+err.Error(), logger.WarnLevel, nil)
	} else if len(netIO) > 0 {
		metrics.NetworkBytesSent = int64(netIO[0].BytesSent)
		metrics.NetworkBytesRecv = int64(netIO[0].BytesRecv)
	}

	return metrics, nil
}
