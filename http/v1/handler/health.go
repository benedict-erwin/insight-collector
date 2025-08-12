package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/benedict-erwin/insight-collector/internal/services/health"
	"github.com/benedict-erwin/insight-collector/pkg/response"
	"github.com/benedict-erwin/insight-collector/pkg/utils"
)

// HealthDetailed returns comprehensive health check information
func HealthDetailed(c echo.Context) error {
	healthStatus, err := health.CheckHealth()
	if err != nil {
		return response.Fail(c, http.StatusInternalServerError, 1, err.Error())
	}

	// Add request ID
	data := map[string]interface{}{
		"health": healthStatus,
	}

	// Return appropriate HTTP status
	httpStatus := http.StatusOK
	switch healthStatus.Status {
	case "unhealthy":
		httpStatus = http.StatusServiceUnavailable
	case "degraded":
		httpStatus = http.StatusPartialContent
	}

	return response.General(c, httpStatus, 0, data, "Health check completed")
}

// HealthLive returns basic liveness check
func HealthLive(c echo.Context) error {
	data := map[string]interface{}{
		"status":    "alive",
		"timestamp": utils.NowFormatted(),
	}

	return response.Success(c, data)
}

// HealthReady returns readiness check for critical services (InfluxDB, Redis, Asynq)
func HealthReady(c echo.Context) error {
	readinessStatus, err := health.CheckReadiness()
	if err != nil {
		return response.Fail(c, http.StatusInternalServerError, 1, err.Error())
	}

	// Return appropriate HTTP status based on readiness
	httpStatus := http.StatusOK
	if readinessStatus.Status != "ready" {
		httpStatus = http.StatusServiceUnavailable
	}

	data := map[string]interface{}{
		"readiness": readinessStatus,
	}

	return response.General(c, httpStatus, 0, data, "Readiness check completed")
}
