package route

import (
	"github.com/labstack/echo/v4"
	"github.com/benedict-erwin/insight-collector/http/registry"
	"github.com/benedict-erwin/insight-collector/http/v1/handler"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
)

func init() {
	// Register example router for v2
	registry.Register("v2", func(g *echo.Group) {
		logger.Info().Msg("Setting up /v2/example routes")
		g.POST("/example", handler.ExamplePost)
	})
}