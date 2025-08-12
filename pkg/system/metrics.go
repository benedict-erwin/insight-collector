package system

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type CPUStat struct {
	User   uint64
	Nice   uint64
	System uint64
	Idle   uint64
	Total  uint64
}

type SystemMetrics struct {
	CPUUsage       float64        `json:"cpu_usage"`
	MemoryUsage    float64        `json:"memory_usage_system"`
	DiskUsage      float64        `json:"disk_usage"`
	GoroutineCount int            `json:"goroutine_count"`
	AppMemory      AppMemoryStats `json:"memory_app"`
}

type AppMemoryStats struct {
	CurrentAlloc string `json:"current_alloc"`
	TotalAlloc   string `json:"total_alloc"`
	SystemMem    string `json:"system_mem"`
	HeapInuse    string `json:"heap_inuse"`
	StackInuse   string `json:"stack_inuse"`
	GCCycles     uint32 `json:"gc_cycles"`
}

// LinuxOnly ensures the application runs only on Linux systems
func LinuxOnly() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("this function only works with linux")
	}
	return nil
}

// getCPUStat reads CPU statistics from /proc/stat (Linux only)
func getCPUStat() (*CPUStat, error) {
	err := LinuxOnly()
	if err != nil {
		return nil, err
	}

	file, err := os.Open("/proc/stat")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return nil, fmt.Errorf("cannot read /proc/stat")
	}

	line := scanner.Text()
	fields := strings.Fields(line)
	if len(fields) < 5 || fields[0] != "cpu" {
		return nil, fmt.Errorf("invalid /proc/stat format")
	}

	user, _ := strconv.ParseUint(fields[1], 10, 64)
	nice, _ := strconv.ParseUint(fields[2], 10, 64)
	system, _ := strconv.ParseUint(fields[3], 10, 64)
	idle, _ := strconv.ParseUint(fields[4], 10, 64)

	total := user + nice + system + idle

	return &CPUStat{
		User:   user,
		Nice:   nice,
		System: system,
		Idle:   idle,
		Total:  total,
	}, nil
}

// getCPUUsage calculates current CPU usage percentage
func getCPUUsage() (float64, error) {
	// Get initial CPU stat
	stat1, err := getCPUStat()
	if err != nil {
		return 0, err
	}

	// Wait 100ms for accurate measurement
	time.Sleep(100 * time.Millisecond)

	// Get second CPU stat
	stat2, err := getCPUStat()
	if err != nil {
		return 0, err
	}

	// Calculate usage
	totalDiff := stat2.Total - stat1.Total
	idleDiff := stat2.Idle - stat1.Idle

	if totalDiff == 0 {
		return 0, nil
	}

	usage := float64(totalDiff-idleDiff) / float64(totalDiff) * 100
	return usage, nil
}

// getMemoryUsage calculates system memory usage percentage from /proc/meminfo
func getMemoryUsage() (float64, error) {
	err := LinuxOnly()
	if err != nil {
		return 0, err
	}
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	var memTotal, memAvailable uint64
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		switch fields[0] {
		case "MemTotal:":
			memTotal, _ = strconv.ParseUint(fields[1], 10, 64)
		case "MemAvailable:":
			memAvailable, _ = strconv.ParseUint(fields[1], 10, 64)
		}

		if memTotal > 0 && memAvailable > 0 {
			break
		}
	}

	if memTotal == 0 {
		return 0, fmt.Errorf("cannot parse memory info")
	}

	memUsed := memTotal - memAvailable
	usage := float64(memUsed) / float64(memTotal) * 100
	return usage, nil
}

// getDiskUsage calculates disk usage percentage for specified path
func getDiskUsage(path string) (float64, error) {
	err := LinuxOnly()
	if err != nil {
		return 0, err
	}

	var stat syscall.Statfs_t
	err = syscall.Statfs(path, &stat)
	if err != nil {
		return 0, err
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	used := total - free

	if total == 0 {
		return 0, nil
	}

	usage := float64(used) / float64(total) * 100
	return usage, nil
}

// GetSystemMetrics collects and returns current system metrics
func GetSystemMetrics() SystemMetrics {
	metrics := SystemMetrics{
		GoroutineCount: runtime.NumGoroutine(),
		AppMemory:      getAppMemoryStats(),
	}

	// CPU Usage
	if cpu, err := getCPUUsage(); err == nil {
		metrics.CPUUsage = cpu
	}

	// Memory Usage
	if mem, err := getMemoryUsage(); err == nil {
		metrics.MemoryUsage = mem
	}

	// Disk Usage (root filesystem)
	if disk, err := getDiskUsage("/"); err == nil {
		metrics.DiskUsage = disk
	}

	return metrics
}

// getAppMemoryStats collects application-specific memory statistics
// getAppMemoryStats collects application-specific memory statistics using Go runtime
func getAppMemoryStats() AppMemoryStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return AppMemoryStats{
		CurrentAlloc: formatBytes(m.Alloc),
		TotalAlloc:   formatBytes(m.TotalAlloc),
		SystemMem:    formatBytes(m.Sys),
		HeapInuse:    formatBytes(m.HeapInuse),
		StackInuse:   formatBytes(m.StackInuse),
		GCCycles:     m.NumGC,
	}
}

// formatBytes converts bytes to human-readable format
// formatBytes converts bytes to human-readable format (B, KB, MB, GB)
func formatBytes(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1fGB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1fMB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1fKB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
