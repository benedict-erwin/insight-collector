package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/benedict-erwin/insight-collector/http/middleware"
	"github.com/benedict-erwin/insight-collector/http/registry"
	"github.com/benedict-erwin/insight-collector/internal/constants"
	asynqPkg "github.com/benedict-erwin/insight-collector/pkg/asynq"
	"github.com/benedict-erwin/insight-collector/pkg/auth"
	"github.com/benedict-erwin/insight-collector/pkg/influxdb"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
	"github.com/benedict-erwin/insight-collector/pkg/maxmind"
	"github.com/benedict-erwin/insight-collector/pkg/redis"
	"github.com/benedict-erwin/insight-collector/pkg/response"
)

// Start initializes and starts the HTTP server
func Start(port int) error {
	// Create new echo instance
	e := echo.New()
	e.HideBanner = true

	// Setup logger scope
	log := logger.WithScope("startServer")

	// Add logger middleware
	e.Use(middleware.Logger)

	// Custom error handler
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		httpStatus := 500
		code := constants.CodeInternalError
		message := constants.GetErrorMessage(code)

		if he, ok := err.(*echo.HTTPError); ok {
			httpStatus = he.Code
			// Map HTTP status to standardized error codes
			switch he.Code {
			case 400:
				code = constants.CodeBadRequest
			case 401:
				code = constants.CodeUnauthorized
			case 403:
				code = constants.CodeForbidden
			case 404:
				code = constants.CodeNotFound
			case 409:
				code = constants.CodeConflict
			case 422:
				code = constants.CodeUnprocessable
			case 429:
				code = constants.CodeRateLimit
			case 500:
				code = constants.CodeInternalError
			case 502:
				code = constants.CodeBadGateway
			case 503:
				code = constants.CodeServiceUnavailable
			case 504:
				code = constants.CodeGatewayTimeout
			default:
				code = constants.CodeInternalError
			}
			
			// Use custom message if provided, otherwise use standard message
			if he.Message != nil {
				message = fmt.Sprintf("%v", he.Message)
			} else {
				message = constants.GetErrorMessage(code)
			}
		}

		if !c.Response().Committed {
			response.Fail(c, httpStatus, code, message)
		}
	}

	// Register routes
	registry.SetupAllRoutes(e)
	log.Info().Interface("routes", e.Routes()).Msg("Registered routes")

	// Start server with graceful shutdown
	go func() {
		addr := fmt.Sprintf(":%d", port)
		log.Info().Msg("Starting server on " + addr)
		if err := e.Start(addr); err != nil {
			log.Error().Err(err).Msg("Server failed to start")
		}
	}()

	// Handle graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server shutdown failed")
		return err
	}

	// Close resources
	influxdb.Close()
	redis.Close()
	maxmind.Close()
	asynqPkg.CloseClient()
	auth.StopAuth()

	// Shutdown completed
	log.Info().Msg("Server gracefully stopped")
	return nil
}
