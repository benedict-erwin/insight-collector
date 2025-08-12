package asynq

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/hibiken/asynq"
	"github.com/benedict-erwin/insight-collector/config"
	"github.com/benedict-erwin/insight-collector/internal/constants"
	"github.com/benedict-erwin/insight-collector/internal/jobs"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
	"github.com/benedict-erwin/insight-collector/pkg/redis"
)

type WorkerConfig struct {
	Name       string   `json:"name"`
	Percentage int      `json:"percentage"`
	TaskTypes  []string `json:"task_types"`
}

var (
	mu                 sync.RWMutex
	currentConcurrency int
	workers            = []WorkerConfig{} // Start with empty workers
	currentServer      *asynq.Server
	serverRunning      bool
)

// InitConcurrency initializes concurrency from config if not set via command
func InitConcurrency() {
	mu.Lock()
	defer mu.Unlock()

	if currentConcurrency == 0 {
		currentConcurrency = config.Get().Asynq.Concurrency
		if currentConcurrency == 0 {
			currentConcurrency = 10 // fallback default
		}
	}

	// Load worker configuration from Redis
	if err := loadWorkersFromRedis(); err != nil {
		logger.Warn().Err(err).Msg("Failed to load worker config from Redis, generating defaults")
	}

	// If no workers loaded, generate defaults from job registry
	if len(workers) == 0 {
		workers = generateDefaultWorkers()
		logger.Info().Int("generated_workers", len(workers)).Msg("Generated default workers from job registry")

		// Save generated defaults to Redis
		if err := saveWorkersToRedis(); err != nil {
			logger.Error().Err(err).Msg("Failed to save generated default workers to Redis")
		}
	}

	logger.Info().Int("concurrency", currentConcurrency).Int("workers_count", len(workers)).Msg("Worker configuration initialized")
}

// GetConcurrency returns current concurrency setting
func GetConcurrency() int {
	mu.RLock()
	defer mu.RUnlock()
	return currentConcurrency
}

// SetConcurrency updates concurrency setting and persists to config file (requires manual restart)
func SetConcurrency(concurrency int) error {
	mu.Lock()
	currentConcurrency = concurrency
	mu.Unlock()

	// Update the config file
	if err := updateConfigFileConcurrency(concurrency); err != nil {
		logger.Error().Err(err).Msg("Failed to update config file with new concurrency")
		return fmt.Errorf("failed to update config file: %w", err)
	}

	logger.Info().Int("new_concurrency", concurrency).Msg("Concurrency updated in memory and config file")
	return nil
}

// SetWorker updates worker configuration and persists to Redis
func SetWorker(name string, percentage int, taskTypes []string) {
	mu.Lock()
	defer mu.Unlock()

	// Find and update existing worker or add new one
	found := false
	for i := range workers {
		if workers[i].Name == name {
			workers[i].Percentage = percentage
			workers[i].TaskTypes = taskTypes
			found = true
			break
		}
	}

	if !found {
		workers = append(workers, WorkerConfig{
			Name:       name,
			Percentage: percentage,
			TaskTypes:  taskTypes,
		})
	}

	// Persist to Redis
	if err := saveWorkersToRedis(); err != nil {
		logger.Error().Err(err).Msg("Failed to save worker config to Redis")
	}

	logger.Info().Str("worker", name).Int("percentage", percentage).Msg("Worker configuration updated and persisted")
}

// GetWorkers returns current worker configurations
func GetWorkers() []WorkerConfig {
	mu.RLock()
	defer mu.RUnlock()

	// Return copy to prevent external modification
	result := make([]WorkerConfig, len(workers))
	copy(result, workers)
	return result
}

// GetWorker returns specific worker configuration
func GetWorker(name string) (WorkerConfig, bool) {
	mu.RLock()
	defer mu.RUnlock()

	for _, worker := range workers {
		if worker.Name == name {
			return worker, true
		}
	}
	return WorkerConfig{}, false
}

// GenerateQueues creates queue config from worker percentages
func GenerateQueues() map[string]int {
	mu.RLock()
	defer mu.RUnlock()

	queues := make(map[string]int)
	for _, worker := range workers {
		queues[worker.Name] = worker.Percentage / 10
		if queues[worker.Name] == 0 {
			queues[worker.Name] = 1 // minimum 1
		}
	}
	return queues
}

// GetQueueForTaskType returns appropriate queue for task type
func GetQueueForTaskType(taskType string) string {
	mu.RLock()
	defer mu.RUnlock()

	for _, worker := range workers {
		for _, t := range worker.TaskTypes {
			if t == taskType {
				return worker.Name
			}
		}
	}
	return "default" // fallback
}

// ValidateWorkerConfig checks if worker configuration is valid
func ValidateWorkerConfig() error {
	mu.RLock()
	defer mu.RUnlock()

	totalPercentage := 0
	taskTypeMap := make(map[string]string)

	for _, worker := range workers {
		totalPercentage += worker.Percentage

		// Check for duplicate task types
		for _, taskType := range worker.TaskTypes {
			if existingWorker, exists := taskTypeMap[taskType]; exists {
				logger.Warn().
					Str("task_type", taskType).
					Str("worker1", existingWorker).
					Str("worker2", worker.Name).
					Msg("Duplicate task type found")
			}
			taskTypeMap[taskType] = worker.Name
		}
	}

	if totalPercentage != 100 {
		logger.Warn().Int("total_percentage", totalPercentage).Msg("Worker percentages do not sum to 100%")
	}

	return nil
}

// ResetToDefault resets worker configuration to empty and persists to Redis
func ResetToDefault() {
	mu.Lock()
	defer mu.Unlock()

	workers = []WorkerConfig{} // Reset to empty

	// Persist empty config to Redis
	if err := saveWorkersToRedis(); err != nil {
		logger.Error().Err(err).Msg("Failed to save reset config to Redis")
	}

	logger.Info().Msg("Worker configuration reset to empty and persisted")
}

// SetCurrentServer stores reference to current server for graceful restart
func SetCurrentServer(server *asynq.Server) {
	mu.Lock()
	defer mu.Unlock()
	currentServer = server
}

// IsServerRunning checks if server is currently running by checking Redis heartbeat
func IsServerRunning() bool {
	mu.RLock()
	defer mu.RUnlock()

	// If in same process, use local flag
	if currentServer != nil && serverRunning {
		return true
	}

	// Check Redis for worker heartbeat from other processes
	return checkWorkerHeartbeat()
}

// SetServerRunning updates server running status
func SetServerRunning(running bool) {
	mu.Lock()
	defer mu.Unlock()
	serverRunning = running
	status := "stopped"
	if running {
		status = "running"
	}
	logger.Info().Bool("running", running).Str("status", status).Msg("Asynq server status updated")
}

// ClearServerReference clears server reference when stopped
func ClearServerReference() {
	mu.Lock()
	defer mu.Unlock()
	currentServer = nil
	serverRunning = false

	// Remove heartbeat from Redis
	removeWorkerHeartbeat()

	logger.Info().Msg("Asynq server reference cleared")
}

// checkWorkerHeartbeat checks if any worker process is running via Redis
func checkWorkerHeartbeat() bool {
	client, err := redis.NewClientForAsynq()
	if err != nil {
		// If Redis client creation fails, assume worker is running to avoid false negatives
		logger.Warn().Err(err).Msg("Cannot create Redis client for heartbeat check")
		return true
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second) // Faster timeout
	defer cancel()

	// Use EXISTS for faster check - key auto-expires if worker is dead
	exists, err := client.Exists(ctx, "asynq:worker:heartbeat")
	if err != nil {
		// If Redis is down, assume worker is running to avoid false negatives
		logger.Warn().Err(err).Msg("Cannot check worker heartbeat - Redis unavailable")
		return true
	}

	return exists
}

// setWorkerHeartbeat sets worker heartbeat in Redis
func setWorkerHeartbeat() {
	client, err := redis.NewClientForAsynq()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create Redis client for heartbeat")
		return
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	timestamp := time.Now().Unix()
	if err := client.Set(ctx, "asynq:worker:heartbeat", timestamp, 60*time.Second); err != nil {
		logger.Error().Err(err).Msg("Failed to set worker heartbeat in Redis")
	}
}

// removeWorkerHeartbeat removes worker heartbeat from Redis
func removeWorkerHeartbeat() {
	client, err := redis.NewClientForAsynq()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create Redis client for heartbeat removal")
		return
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Delete(ctx, "asynq:worker:heartbeat"); err != nil {
		logger.Error().Err(err).Msg("Failed to remove worker heartbeat from Redis")
	}
}

// SetWorkerHeartbeat is public wrapper for setWorkerHeartbeat
func SetWorkerHeartbeat() {
	setWorkerHeartbeat()
}

// saveWorkersToRedis persists worker configuration to Redis
func saveWorkersToRedis() error {
	client, err := redis.NewClientForAsynq()
	if err != nil {
		return fmt.Errorf("failed to create Redis client for worker config: %w", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = client.SetJSON(ctx, "asynq:worker:config", workers, 0)
	if err != nil {
		return fmt.Errorf("failed to save worker config to Redis: %w", err)
	}

	logger.Debug().Int("workers_count", len(workers)).Msg("Worker configuration saved to Redis")
	return nil
}

// loadWorkersFromRedis loads worker configuration from Redis
func loadWorkersFromRedis() error {
	client, err := redis.NewClientForAsynq()
	if err != nil {
		return fmt.Errorf("failed to create Redis client for worker config: %w", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to get JSON data
	err = client.GetJSON(ctx, "asynq:worker:config", &workers)
	if err != nil {
		// Check if key doesn't exist
		exists, existsErr := client.Exists(ctx, "asynq:worker:config")
		if existsErr == nil && !exists {
			// No config in Redis, keep empty workers
			logger.Info().Msg("No worker configuration found in Redis, starting with empty config")
			return nil
		}
		return fmt.Errorf("failed to load worker config from Redis: %w", err)
	}

	logger.Info().Int("workers_count", len(workers)).Msg("Worker configuration loaded from Redis")
	return nil
}

// generateDefaultWorkers creates worker configuration from job registry
func generateDefaultWorkers() []WorkerConfig {
	registeredJobs, err := jobs.GetRegisteredJobs()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get registered jobs, creating empty worker config")
		return []WorkerConfig{}
	}

	if len(registeredJobs) == 0 {
		logger.Warn().Msg("No registered jobs found, creating empty worker config")
		return []WorkerConfig{}
	}

	// Group jobs by queue
	queueJobs := make(map[string][]string)
	for _, job := range registeredJobs {
		queueJobs[job.Queue] = append(queueJobs[job.Queue], job.TaskType)
	}

	// Create workers with smart percentages
	var defaultWorkers []WorkerConfig
	for queue, taskTypes := range queueJobs {
		percentage := getDefaultPercentage(queue, len(queueJobs))
		defaultWorkers = append(defaultWorkers, WorkerConfig{
			Name:       queue,
			Percentage: percentage,
			TaskTypes:  taskTypes,
		})
	}

	// Normalize percentages to sum to 100%
	defaultWorkers = normalizePercentages(defaultWorkers)

	logger.Info().
		Int("total_jobs", len(registeredJobs)).
		Int("total_queues", len(queueJobs)).
		Interface("queue_distribution", queueJobs).
		Msg("Generated default worker configuration from job registry")

	return defaultWorkers
}

// getDefaultPercentage returns smart default percentage for queue
func getDefaultPercentage(queue string, totalQueues int) int {
	switch queue {
	case constants.QueueCritical:
		return 60 // High priority jobs get most resources
	case constants.QueueDefault:
		return 30 // Normal jobs get standard allocation
	case constants.QueueLow:
		return 10 // Background jobs get minimal resources
	default:
		// For custom queues, distribute remaining percentage evenly
		return 100 / totalQueues
	}
}

// normalizePercentages ensures percentages sum to 100%
func normalizePercentages(workers []WorkerConfig) []WorkerConfig {
	if len(workers) == 0 {
		return workers
	}

	totalPercentage := 0
	for _, worker := range workers {
		totalPercentage += worker.Percentage
	}

	if totalPercentage == 0 {
		// If all percentages are 0, distribute evenly
		evenPercentage := 100 / len(workers)
		remainder := 100 % len(workers)

		for i := range workers {
			workers[i].Percentage = evenPercentage
			if i < remainder {
				workers[i].Percentage++
			}
		}
	} else if totalPercentage != 100 {
		// Adjust percentages proportionally to sum to 100%
		for i := range workers {
			workers[i].Percentage = (workers[i].Percentage * 100) / totalPercentage
		}

		// Handle rounding errors - ensure exactly 100%
		actualTotal := 0
		for _, worker := range workers {
			actualTotal += worker.Percentage
		}

		if actualTotal != 100 && len(workers) > 0 {
			// Add/subtract difference to first worker
			workers[0].Percentage += (100 - actualTotal)
		}
	}

	return workers
}

// updateConfigFileConcurrency updates the concurrency value in the config file
func updateConfigFileConcurrency(concurrency int) error {
	const configFile = ".config.json"
	
	// Read current config file
	configData, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	
	// Parse JSON
	var configMap map[string]interface{}
	if err := json.Unmarshal(configData, &configMap); err != nil {
		return fmt.Errorf("failed to parse config JSON: %w", err)
	}
	
	// Update asynq.concurrency field
	if asynqConfig, exists := configMap["asynq"]; exists {
		if asynqMap, ok := asynqConfig.(map[string]interface{}); ok {
			asynqMap["concurrency"] = concurrency
		} else {
			return fmt.Errorf("asynq config is not a valid object")
		}
	} else {
		// Create asynq config if it doesn't exist
		configMap["asynq"] = map[string]interface{}{
			"concurrency": concurrency,
			"db":          0, // default db
		}
	}
	
	// Write back to file with proper formatting
	updatedData, err := json.MarshalIndent(configMap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated config: %w", err)
	}
	
	if err := os.WriteFile(configFile, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	logger.Info().Int("concurrency", concurrency).Str("config_file", configFile).Msg("Configuration file updated with new concurrency")
	return nil
}
