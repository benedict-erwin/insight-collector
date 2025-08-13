package cmd

import (
	"os"

	"github.com/benedict-erwin/insight-collector/config"
	asynqPkg "github.com/benedict-erwin/insight-collector/pkg/asynq"
	"github.com/benedict-erwin/insight-collector/pkg/auth"
	"github.com/benedict-erwin/insight-collector/pkg/influxdb"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
	"github.com/benedict-erwin/insight-collector/pkg/maxmind"
	"github.com/benedict-erwin/insight-collector/pkg/redis"
	"github.com/benedict-erwin/insight-collector/pkg/utils"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "insight-collector",
	Short: "InsightCollector HTTP Service",
	Long:  `InsightCollector HTTP Service for storing log and metric data`,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logger.Error().Err(err).Msg("Failed to execute command")
		os.Exit(1)
	}
}

// init initializes all application dependencies and registers commands
func init() {
	// Initialize config
	if err := config.Init(); err != nil {
		panic(err)
	}

	// Initialize logger
	logger.Init(config.Get().App.Timezone, config.Get().App.Env)

	// Initialize Redis
	if err := redis.Init(); err != nil {
		logger.Error().Err(err).Msg("Failed to initialize Redis")
		panic(err)
	}

	// Initialize InfluxDB
	if err := influxdb.Init(); err != nil {
		logger.Error().Err(err).Msg("Failed to initialize InfluxDB")
		panic(err)
	}

	// Initialize utils
	if err := utils.InitTimezone(); err != nil {
		logger.Warn().Err(err).Msg("Timezone initialization failed, continuing with UTC")
		panic(err)
	}

	// Initialize asynq client
	if err := asynqPkg.InitClient(); err != nil {
		logger.Error().Err(err).Msg("Failed to initialize Asynq client")
		// Continue without queue functionality
	}

	// Initialize worker configuration (loads from Redis)
	asynqPkg.InitConcurrency()

	// Initialize auth system
	if err := auth.InitAuth(); err != nil {
		logger.Error().Err(err).Msg("Failed to initialize Auth system")
		panic(err)
	}

	// Initialize MaxMind GeoIP service
	if err := maxmind.Init(); err != nil {
		logger.Warn().Err(err).Msg("MaxMind GeoIP service failed to start, will use fallback")
		// Continue without panicking - service handles fallback gracefully
	}

	// User agent (disabled)
	// useragent.Init()

	// Add commands
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(devCmd)
	rootCmd.AddCommand(workerCmd)
}
