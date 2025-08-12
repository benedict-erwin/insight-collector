package cmd

import (
	"github.com/benedict-erwin/insight-collector/config"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
	"github.com/benedict-erwin/insight-collector/server"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start HTTP Server",
	Long:  `Starts the InsightCollector HTTP Server`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := server.Start(config.Get().App.Port); err != nil {
			logger.WithScope("serveCmd").Error().Err(err).Msg("Failed to start server")
		}
	},
}
