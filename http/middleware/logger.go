package middleware

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/benedict-erwin/insight-collector/internal/constants"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
	"github.com/benedict-erwin/insight-collector/pkg/utils"
)

// Logger middleware logs HTTP requests with timing and generates request IDs
func Logger(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// start timer
		start := utils.Now()

		// Get Request ID from header or generate it
		reqId := constants.GetRequestIDFromHeaders(c)
		if reqId == "" {
			reqId = string(generateRequestID())
		}

		// Save the request id in context
		c.Set(constants.RequestIDKey, reqId)

		// Execute Handler
		err := next(c)

		// Count latency
		latency := time.Since(start).Microseconds()

		// Get HTTP status
		status := c.Response().Status

		// Check for error
		if err != nil {
			if he, ok := err.(*echo.HTTPError); ok {
				status = he.Code
			}
		}

		// Request logger
		log := logger.WithScope("accessLog")
		log.Info().
			Str("method", c.Request().Method).
			Str("path", c.Request().URL.Path).
			Int("status", status).
			Int64("latency", latency).
			Str("request-id", reqId).
			Msg("HTTP Request")

		return err
	}
}

// generateRequestID creates unique request identifier with timestamp and random component
func generateRequestID() string {
	timestamp := utils.Now().Unix()
	random := rand.Uint32()
	return fmt.Sprintf("req-%d-%08x", timestamp, random)
}
