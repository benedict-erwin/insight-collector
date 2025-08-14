package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	"github.com/labstack/echo/v4"
)

// Custom http server
var httpServer *http.Server

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

	// Log registered routes
	log.Info().Interface("routes", e.Routes()).Msg("Registered routes")

	// For development only
	if os.Getenv("GODEBUG") != "" {
		// Add connection monitoring endpoint
		e.GET("/debug/connections", func(c echo.Context) error {
			stats := map[string]interface{}{
				"timestamp": time.Now().Format(time.RFC3339),
				"server_info": map[string]interface{}{
					"read_timeout":         "10s",
					"write_timeout":        "10s",
					"idle_timeout":         "30s",
					"read_header_timeout":  "5s",
					"max_header_bytes":     1048576,
					"keepalive_period":     "15s",
					"tcp_nodelay":          true,
					"tcp_linger":           0,
				},
			}
			return response.Success(c, stats)
		})

		// Start pprof server for profiling
		go func() {
			pprofAddr := fmt.Sprintf(":%d", port+1000)
			log.Info().Msg("Starting pprof server on " + pprofAddr)
			if err := http.ListenAndServe(pprofAddr, nil); err != nil {
				log.Error().Err(err).Msg("Pprof server failed to start")
			}
		}()
	}

	// Configure HTTP server with optimized settings for high load
	addr := fmt.Sprintf(":%d", port)
	httpServer = &http.Server{
		Addr:              addr,
		Handler:           e,
		ReadTimeout:       10 * time.Second, // Reduced for faster timeout detection
		WriteTimeout:      10 * time.Second, // Reduced for faster timeout detection
		IdleTimeout:       30 * time.Second, // Reduced to free connections faster
		ReadHeaderTimeout: 5 * time.Second,  // Reduced for faster header processing
		MaxHeaderBytes:    1 << 20,          // 1MB

		// HIGH LOAD OPTIMIZATIONS
		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
			// Advanced TCP socket optimizations
			if tcpConn, ok := c.(*net.TCPConn); ok {
				tcpConn.SetKeepAlive(true)
				tcpConn.SetKeepAlivePeriod(15 * time.Second) // More aggressive keepalive
				tcpConn.SetNoDelay(true)                     // Disable Nagle's algorithm for lower latency
				tcpConn.SetLinger(0)                         // Close immediately without waiting
			}
			return ctx
		},

		// Connection state monitoring for high load debugging
		ConnState: func(conn net.Conn, state http.ConnState) {
			// Log connection state changes during high load
			if state == http.StateNew || state == http.StateClosed {
				// Only log during development/debugging
				if os.Getenv("GODEBUG") != "" {
					log.Debug().
						Str("remote_addr", conn.RemoteAddr().String()).
						Str("state", state.String()).
						Msg("HTTP connection state changed")
				}
			}
		},
	}

	// Start server with graceful shutdown
	go func() {
		log.Info().
			Str("addr", addr).
			Dur("read_timeout", 10*time.Second).
			Dur("write_timeout", 10*time.Second).
			Dur("idle_timeout", 30*time.Second).
			Dur("keepalive_period", 15*time.Second).
			Bool("tcp_nodelay", true).
			Msg("Starting HTTP server with advanced TCP optimization")

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("HTTP server failed to start")
		}
	}()

	// Handle graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
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
