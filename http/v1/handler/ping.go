package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	pingEntity "github.com/benedict-erwin/insight-collector/internal/entities/ping"
	"github.com/benedict-erwin/insight-collector/internal/services/ping"
	"github.com/benedict-erwin/insight-collector/pkg/response"
)

// Ping handles GET ping requests
func Ping(c echo.Context) error {
	data := map[string]string{
		"responses": ping.Ping(),
	}
	return response.Success(c, data)
}

// PingPost handles POST ping requests with validation
func PingPost(c echo.Context) error {
	var req pingEntity.PingRequest
	if err := c.Bind(&req); err != nil {
		return response.Fail(c, http.StatusBadRequest, 1, "Invalid JSON payload")
	}

	result, err := ping.PingPost(req)
	if err != nil {
		return response.Fail(c, http.StatusBadRequest, 1, err.Error())
	}

	data := map[string]string{
		"responses": result,
	}
	return response.Success(c, data)
}
