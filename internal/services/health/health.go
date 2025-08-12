package health

import (
	"fmt"
	"sync"
	"time"

	"github.com/benedict-erwin/insight-collector/config"
	"github.com/benedict-erwin/insight-collector/pkg/asynq"
	"github.com/benedict-erwin/insight-collector/pkg/influxdb"
	"github.com/benedict-erwin/insight-collector/pkg/maxmind"
	"github.com/benedict-erwin/insight-collector/pkg/redis"
	"github.com/benedict-erwin/insight-collector/pkg/system"
	"github.com/benedict-erwin/insight-collector/pkg/utils"
)

var (
	startTime = time.Now()
	
	// Cache for health checks
	healthCache      *HealthStatus
	healthCacheTime  time.Time
	healthCacheMutex sync.RWMutex
	
	readinessCache      *ReadinessStatus
	readinessCacheTime  time.Time
	readinessCacheMutex sync.RWMutex
	
	cacheValidDuration = 10 * time.Second
)

type HealthStatus struct {
	Status    string                   `json:"status"`
	Timestamp time.Time                `json:"timestamp"`
	Version   string                   `json:"version"`
	Uptime    string                   `json:"uptime"`
	Services  map[string]ServiceHealth `json:"services"`
	System    SystemHealth             `json:"system"`
}

type ServiceHealth struct {
	Status       string                 `json:"status"`
	ResponseTime string                 `json:"response_time"`
	LastCheck    time.Time              `json:"last_check"`
	Error        string                 `json:"error,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type SystemHealth struct {
	MemoryUsageSystem string                `json:"memory_usage_system"`
	MemoryApp         system.AppMemoryStats `json:"memory_app"`
	CPUUsage          string                `json:"cpu_usage"`
	DiskUsage         string                `json:"disk_usage"`
	GoroutineCount    int                   `json:"goroutine_count"`
}

type ReadinessStatus struct {
	Status    string                   `json:"status"`
	Timestamp time.Time                `json:"timestamp"`
	Services  map[string]ServiceHealth `json:"services"`
}

// CheckHealth performs comprehensive health checks and returns status with 10s cache
func CheckHealth() (*HealthStatus, error) {
	// Check cache first
	healthCacheMutex.RLock()
	if healthCache != nil && time.Since(healthCacheTime) < cacheValidDuration {
		cached := *healthCache // Copy to avoid race conditions
		healthCacheMutex.RUnlock()
		return &cached, nil
	}
	healthCacheMutex.RUnlock()

	// Cache miss - perform actual health check
	cfg := config.Get()

	status := &HealthStatus{
		Timestamp: time.Now(),
		Version:   cfg.App.Version,
		Uptime:    time.Since(startTime).String(),
		Services:  make(map[string]ServiceHealth),
		System:    getSystemMetrics(),
	}

	overallHealthy := true

	// Check InfluxDB
	influxHealth := checkInfluxDB()
	status.Services["influxdb"] = influxHealth
	if influxHealth.Status != "healthy" {
		overallHealthy = false
	}

	// Check Redis
	redisHealth := checkRedis()
	status.Services["redis"] = redisHealth
	if redisHealth.Status != "healthy" {
		overallHealthy = false
	}

	// Check Asynq
	asynqHealth := checkAsynq()
	status.Services["asynq"] = asynqHealth
	if asynqHealth.Status != "healthy" {
		overallHealthy = false
	}

	// Check MaxMind
	maxmindHealth := checkMaxMind()
	status.Services["maxmind"] = maxmindHealth
	// Note: MaxMind degraded state doesn't affect overall health
	// since it has fallback behavior

	// Determine overall status
	if overallHealthy {
		status.Status = "healthy"
	} else {
		status.Status = "degraded"
	}

	// Update cache
	healthCacheMutex.Lock()
	healthCache = status
	healthCacheTime = time.Now()
	healthCacheMutex.Unlock()

	return status, nil
}

// checkInfluxDB performs InfluxDB connectivity and health check
func checkInfluxDB() ServiceHealth {
	start := utils.Now()

	// Check if client exists
	if !influxdb.IsHealthy() {
		return ServiceHealth{
			Status:       "unhealthy",
			ResponseTime: "0ms",
			LastCheck:    utils.Now(),
			Error:        "InfluxDB client not initialized",
		}
	}

	// Use the health check function
	err := influxdb.HealthCheck()
	responseTime := time.Since(start)

	if err != nil {
		return ServiceHealth{
			Status:       "unhealthy",
			ResponseTime: responseTime.String(),
			LastCheck:    utils.Now(),
			Error:        err.Error(),
		}
	}

	return ServiceHealth{
		Status:       "healthy",
		ResponseTime: responseTime.String(),
		LastCheck:    utils.Now(),
	}
}

// checkRedis performs Redis connectivity check
func checkRedis() ServiceHealth {
	start := utils.Now()

	// Use centralized Redis health check
	err := redis.Health()
	responseTime := time.Since(start)

	if err != nil {
		return ServiceHealth{
			Status:       "unhealthy",
			ResponseTime: responseTime.String(),
			LastCheck:    utils.Now(),
			Error:        err.Error(),
		}
	}

	return ServiceHealth{
		Status:       "healthy",
		ResponseTime: responseTime.String(),
		LastCheck:    utils.Now(),
	}
}

// checkAsynq performs Asynq service check
func checkAsynq() ServiceHealth {
	start := utils.Now()

	// Check if Asynq client is initialized
	client := asynq.GetClient()
	if client == nil {
		return ServiceHealth{
			Status:       "unhealthy",
			ResponseTime: "0ms",
			LastCheck:    utils.Now(),
			Error:        "Asynq client not initialized",
		}
	}

	// Check if server is running
	serverRunning := asynq.IsServerRunning()
	responseTime := time.Since(start)

	status := "healthy"
	errorMsg := ""

	if !serverRunning {
		status = "degraded"
		errorMsg = "Asynq server not running"
	}

	return ServiceHealth{
		Status:       status,
		ResponseTime: responseTime.String(),
		LastCheck:    utils.Now(),
		Error:        errorMsg,
	}
}

// checkMaxMind performs MaxMind GeoIP service check
func checkMaxMind() ServiceHealth {
	start := utils.Now()

	// Get database information
	dbInfo := maxmind.GetDatabaseInfo()
	responseTime := time.Since(start)

	status := "healthy"
	errorMsg := ""
	metadata := make(map[string]interface{})

	if !dbInfo.Enabled {
		status = "disabled"
		errorMsg = "MaxMind service is disabled"
	} else if dbInfo.LoadedAt.IsZero() {
		status = "degraded"
		errorMsg = "GeoIP databases not loaded, using fallback"
	} else {
		// Service is working, populate metadata
		metadata["database_version"] = dbInfo.CityDBModTime.Format("2006-01-02")
		metadata["last_reload"] = dbInfo.LoadedAt.Format("2006-01-02 15:04:05")
		metadata["reload_count"] = dbInfo.ReloadCount
		
		// Check if database files are recent (within 2 weeks for weekly updates)
		if time.Since(dbInfo.CityDBModTime) > 14*24*time.Hour {
			status = "degraded"
			errorMsg = "GeoIP database appears outdated"
		}
	}

	return ServiceHealth{
		Status:       status,
		ResponseTime: responseTime.String(),
		LastCheck:    utils.Now(),
		Error:        errorMsg,
		Metadata:     metadata,
	}
}

// getSystemMetrics collects current system performance metrics
func getSystemMetrics() SystemHealth {
	// Get system metrics using pure Go implementation
	metrics := system.GetSystemMetrics()

	return SystemHealth{
		CPUUsage:          fmt.Sprintf("%.1f%%", metrics.CPUUsage),
		MemoryUsageSystem: fmt.Sprintf("%.1f%%", metrics.MemoryUsage),
		MemoryApp:         metrics.AppMemory,
		DiskUsage:         fmt.Sprintf("%.1f%%", metrics.DiskUsage),
		GoroutineCount:    metrics.GoroutineCount,
	}
}

// CheckReadiness performs readiness checks for critical services with 10s cache
func CheckReadiness() (*ReadinessStatus, error) {
	// Check cache first
	readinessCacheMutex.RLock()
	if readinessCache != nil && time.Since(readinessCacheTime) < cacheValidDuration {
		cached := *readinessCache // Copy to avoid race conditions
		readinessCacheMutex.RUnlock()
		return &cached, nil
	}
	readinessCacheMutex.RUnlock()

	// Cache miss - perform actual readiness check
	status := &ReadinessStatus{
		Timestamp: time.Now(),
		Services:  make(map[string]ServiceHealth),
	}

	overallReady := true

	// Check InfluxDB - critical for logging
	influxHealth := checkInfluxDB()
	status.Services["influxdb"] = influxHealth
	if influxHealth.Status != "healthy" {
		overallReady = false
	}

	// Check Redis - critical for job queue
	redisHealth := checkRedis()
	status.Services["redis"] = redisHealth
	if redisHealth.Status != "healthy" {
		overallReady = false
	}

	// Check Asynq - critical for background jobs
	asynqHealth := checkAsynq()
	status.Services["asynq"] = asynqHealth
	if asynqHealth.Status != "healthy" {
		overallReady = false
	}

	// Determine overall readiness
	if overallReady {
		status.Status = "ready"
	} else {
		status.Status = "not_ready"
	}

	// Update cache
	readinessCacheMutex.Lock()
	readinessCache = status
	readinessCacheTime = time.Now()
	readinessCacheMutex.Unlock()

	return status, nil
}

// ClearHealthCache clears health check cache (useful for testing/debugging)
func ClearHealthCache() {
	healthCacheMutex.Lock()
	healthCache = nil
	healthCacheTime = time.Time{}
	healthCacheMutex.Unlock()
}

// ClearReadinessCache clears readiness check cache (useful for testing/debugging)
func ClearReadinessCache() {
	readinessCacheMutex.Lock()
	readinessCache = nil
	readinessCacheTime = time.Time{}
	readinessCacheMutex.Unlock()
}
