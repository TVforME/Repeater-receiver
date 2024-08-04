package systemstats

import (
	"fmt"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
)

// SystemStats holds the system statistics
type SystemStats struct {
	CPU      float64 `json:"cpu"`
	RAM      float64 `json:"ram"`
	RAMTotal float64 `json:"ramTotal"`
	Network  string  `json:"network"`
}

// formatBytes converts bytes to a human-readable format
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// getSystemStats gathers the system statistics
func GetSystemStats() (SystemStats, error) {
	cpuPercent, err := cpu.Percent(0, false)
	if err != nil {
		return SystemStats{}, err
	}

	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return SystemStats{}, err
	}

	netIO, err := net.IOCounters(false)
	if err != nil {
		return SystemStats{}, err
	}

	network := "N/A"
	if len(netIO) > 0 {
		network = fmt.Sprintf("Sent: %v, Received: %v", formatBytes(netIO[0].BytesSent), formatBytes(netIO[0].BytesRecv))
	}

	return SystemStats{
		CPU:      cpuPercent[0],
		RAM:      vmStat.UsedPercent,
		RAMTotal: float64(vmStat.Total) / (1024 * 1024 * 1024), // Convert total RAM to GB
		Network:  network,
	}, nil
}
