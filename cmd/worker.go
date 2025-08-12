package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/spf13/cobra"

	"github.com/benedict-erwin/insight-collector/internal/jobs"
	asynqPkg "github.com/benedict-erwin/insight-collector/pkg/asynq"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
	"github.com/benedict-erwin/insight-collector/pkg/utils"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Manage background job workers",
	Long:  `Manage Asynq background job workers and configuration`,
}

// Subcommands
var (
	workerStartCmd = &cobra.Command{
		Use:   "start",
		Short: "Start background job worker",
		Long:  `Start Asynq worker to process background jobs`,
		Run: func(cmd *cobra.Command, args []string) {
			startWorker()
		},
	}

	workerListCmd = &cobra.Command{
		Use:   "list",
		Short: "List worker configurations",
		Run: func(cmd *cobra.Command, args []string) {
			listWorkers()
		},
	}

	workerShowCmd = &cobra.Command{
		Use:   "show [worker-name]",
		Short: "Show specific worker configuration",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			showWorker(args[0])
		},
	}

	workerSetCmd = &cobra.Command{
		Use:   "set [worker-name] [percentage] [task-types]",
		Short: "Set worker configuration",
		Long:  `Set worker percentage and task types. Task types should be comma-separated.`,
		Args:  cobra.ExactArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			setWorker(args[0], args[1], args[2])
		},
	}

	workerAddCmd = &cobra.Command{
		Use:   "add [worker-name] [task-types]",
		Short: "Add task types to existing worker",
		Long:  `Add new task types to existing worker without changing existing ones. Task types should be comma-separated.`,
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			addWorkerTaskTypes(args[0], args[1])
		},
	}

	workerStatusCmd = &cobra.Command{
		Use:   "status",
		Short: "Show current queue status",
		Run: func(cmd *cobra.Command, args []string) {
			showStatus()
		},
	}

	workerValidateCmd = &cobra.Command{
		Use:   "validate",
		Short: "Validate worker configuration",
		Run: func(cmd *cobra.Command, args []string) {
			validateConfig()
		},
	}

	workerResetCmd = &cobra.Command{
		Use:   "reset",
		Short: "Reset to default configuration",
		Run: func(cmd *cobra.Command, args []string) {
			resetConfig()
		},
	}

	workerConcurrencyCmd = &cobra.Command{
		Use:   "concurrency [number]",
		Short: "Set worker concurrency",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			setConcurrency(args[0])
		},
	}
)

// startWorker initializes and starts the Asynq worker server with graceful shutdown
func startWorker() {
	// Setup logger scope
	log := logger.WithScope("startWorker")

	// Initialize asynq client
	if err := asynqPkg.InitClient(); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Asynq client")
	}

	// Initialize server
	server := asynqPkg.InitServer()
	mux := asynq.NewServeMux()

	// Register handlers (ignore returned job metadata in worker context)
	_, err := jobs.RegisterHandlers(mux)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to register job handlers")
	}

	// Setup shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start heartbeat goroutine
	go func() {
		ticker := time.NewTicker(15 * time.Second) // Reduced to 15 seconds for less overhead
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				asynqPkg.SetWorkerHeartbeat()
			case <-sigChan:
				return
			}
		}
	}()

	// Start server
	go func() {
		log.Info().Msg("Starting Asynq worker server...")
		asynqPkg.SetServerRunning(true) // Mark server as running
		asynqPkg.SetWorkerHeartbeat()   // Send initial heartbeat

		if err := server.Run(mux); err != nil {
			asynqPkg.SetServerRunning(false) // Mark server as stopped on error
			log.Fatal().Err(err).Msg("Failed to start worker server")
		}
		asynqPkg.SetServerRunning(false) // Mark server as stopped when Run() exits
	}()

	// Wait for shutdown signal
	sig := <-sigChan
	log.Info().Str("signal", sig.String()).Msg("Received shutdown signal, initiating graceful shutdown...")

	log.Info().Msg("Stopping server, waiting for running tasks to complete (max 30s)...")

	// Shutdown waits for tasks to finish
	// Timeout: 30 seconds
	server.Shutdown()

	// Clear server reference and status
	asynqPkg.ClearServerReference()

	log.Info().Msg("Worker server stopped gracefully - all tasks completed or timed out")
}

// listWorkers displays all worker configurations in a table format
func listWorkers() {
	// Initialize concurrency and load config from Redis
	asynqPkg.InitConcurrency()

	workers := asynqPkg.GetWorkers()
	queues := asynqPkg.GenerateQueues()
	concurrency := asynqPkg.GetConcurrency()

	// Prepare JSON output structure
	totalWeight := 0
	for _, count := range queues {
		totalWeight += count
	}

	output := map[string]interface{}{
		"concurrency":   concurrency,
		"total_weight":  totalWeight,
		"workers_count": len(workers),
		"workers":       []map[string]interface{}{},
	}

	// Build workers data
	for _, worker := range workers {
		count := queues[worker.Name]
		workerData := map[string]interface{}{
			"queue":      worker.Name,
			"percentage": worker.Percentage,
			"count":      count,
			"task_types": worker.TaskTypes,
		}
		output["workers"] = append(output["workers"].([]map[string]interface{}), workerData)
	}

	// Output as pretty JSON
	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return
	}

	utils.ClearScreen()
	fmt.Println(string(jsonData))
}

// showWorker displays detailed configuration for a specific worker in JSON format
func showWorker(name string) {
	// Initialize concurrency and load config from Redis
	asynqPkg.InitConcurrency()

	worker, found := asynqPkg.GetWorker(name)
	if !found {
		fmt.Printf("{\"error\": \"Worker '%s' not found\"}\n", name)
		return
	}

	queues := asynqPkg.GenerateQueues()
	count := queues[worker.Name]

	// Prepare JSON output
	output := map[string]interface{}{
		"queue":      worker.Name,
		"percentage": worker.Percentage,
		"count":      count,
		"task_types": worker.TaskTypes,
	}

	// Output as pretty JSON
	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Printf("{\"error\": \"Failed to marshal JSON: %v\"}\n", err)
		return
	}

	utils.ClearScreen()
	fmt.Println(string(jsonData))
}

// setWorker updates worker configuration with new percentage and task types
func setWorker(name, percentageStr, taskTypesStr string) {
	// Initialize concurrency and load config from Redis
	asynqPkg.InitConcurrency()

	percentage, err := strconv.Atoi(percentageStr)
	if err != nil {
		fmt.Printf("Invalid percentage: %s\n", percentageStr)
		return
	}

	if percentage < 0 || percentage > 100 {
		fmt.Printf("Percentage must be between 0 and 100\n")
		return
	}

	taskTypes := []string{}
	if taskTypesStr != "" {
		taskTypes = strings.Split(taskTypesStr, ",")
		for i := range taskTypes {
			taskTypes[i] = strings.TrimSpace(taskTypes[i])
		}
	}

	asynqPkg.SetWorker(name, percentage, taskTypes)
	fmt.Printf("Worker '%s' updated: %d%% with %d task types\n", name, percentage, len(taskTypes))
}

// addWorkerTaskTypes adds new task types to existing worker without changing existing ones
func addWorkerTaskTypes(name, taskTypesStr string) {
	// Initialize concurrency and load config from Redis
	asynqPkg.InitConcurrency()

	// Get current worker configuration
	currentWorker, exists := asynqPkg.GetWorker(name)
	if !exists {
		fmt.Printf("Worker '%s' not found. Available workers:\n", name)
		listWorkers()
		return
	}

	// Parse new task types
	newTaskTypes := []string{}
	if taskTypesStr != "" {
		newTaskTypes = strings.Split(taskTypesStr, ",")
		for i := range newTaskTypes {
			newTaskTypes[i] = strings.TrimSpace(newTaskTypes[i])
		}
	}

	// Merge existing and new task types (avoid duplicates)
	taskTypeMap := make(map[string]bool)
	for _, taskType := range currentWorker.TaskTypes {
		taskTypeMap[taskType] = true
	}

	addedCount := 0
	for _, taskType := range newTaskTypes {
		if !taskTypeMap[taskType] {
			currentWorker.TaskTypes = append(currentWorker.TaskTypes, taskType)
			taskTypeMap[taskType] = true
			addedCount++
		}
	}

	if addedCount == 0 {
		fmt.Printf("No new task types added to worker '%s' (all already exist)\n", name)
		return
	}

	// Update worker with merged task types (keep same percentage)
	asynqPkg.SetWorker(name, currentWorker.Percentage, currentWorker.TaskTypes)
	fmt.Printf("Worker '%s': added %d new task types (total: %d task types)\n", name, addedCount, len(currentWorker.TaskTypes))
}

// showStatus displays current queue configuration and concurrency settings
func showStatus() {
	// Initialize concurrency and load config from Redis
	asynqPkg.InitConcurrency()

	queues := asynqPkg.GenerateQueues()
	concurrency := asynqPkg.GetConcurrency()

	utils.ClearScreen()
	fmt.Printf("Active Queue Configuration:\n")
	total := 0
	for queueName, weight := range queues {
		fmt.Printf("  %s: %d weight\n", queueName, weight)
		total += weight
	}
	fmt.Printf("\nConcurrency: %d workers\n", concurrency)
	fmt.Printf("Total Weight: %d\n", total)
}

// validateConfig checks if current worker configuration is valid
func validateConfig() {
	// Initialize concurrency and load config from Redis
	asynqPkg.InitConcurrency()

	if err := asynqPkg.ValidateWorkerConfig(); err != nil {
		fmt.Printf("Validation failed: %v\n", err)
		return
	}
	fmt.Println("Worker configuration is valid")
}

// resetConfig resets worker configuration to default values
func resetConfig() {
	// Initialize concurrency and load config from Redis first
	asynqPkg.InitConcurrency()

	asynqPkg.ResetToDefault()
	fmt.Println("Worker configuration reset to empty")
}

// setConcurrency updates worker concurrency setting with restart instructions
func setConcurrency(concurrencyStr string) {
	concurrency, err := strconv.Atoi(concurrencyStr)
	if err != nil {
		fmt.Printf("Invalid concurrency: %s\n", concurrencyStr)
		return
	}

	if concurrency < 1 {
		fmt.Printf("Concurrency must be at least 1\n")
		return
	}

	if err := asynqPkg.SetConcurrency(concurrency); err != nil {
		fmt.Printf("Failed to set concurrency: %v\n", err)
		return
	}

	fmt.Printf("✅ Concurrency updated to %d\n", concurrency)

	if asynqPkg.IsServerRunning() {
		fmt.Println("⚠️  Worker is currently running. Please restart worker to apply new concurrency:")
		fmt.Println("   1. Stop current worker gracefully (Ctrl+C)")
		fmt.Println("      - Worker will wait for running tasks to complete (max 30s)")
		fmt.Println("   2. Run: ./app worker start")
		fmt.Println("")
		fmt.Println("ℹ️  Note: Ctrl+C is now safe - it triggers graceful shutdown")
	} else {
		fmt.Println("💡 Start worker with: ./app worker start")
	}
}

// init registers all worker subcommands with the root command
func init() {
	// Register subcommands
	workerCmd.AddCommand(workerStartCmd)
	workerCmd.AddCommand(workerListCmd)
	workerCmd.AddCommand(workerShowCmd)
	workerCmd.AddCommand(workerSetCmd)
	workerCmd.AddCommand(workerAddCmd)
	workerCmd.AddCommand(workerStatusCmd)
	workerCmd.AddCommand(workerValidateCmd)
	workerCmd.AddCommand(workerResetCmd)
	workerCmd.AddCommand(workerConcurrencyCmd)

	// Register worker command
	rootCmd.AddCommand(workerCmd)
}
