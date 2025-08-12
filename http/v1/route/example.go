package route

import (
	"github.com/labstack/echo/v4"
	"github.com/benedict-erwin/insight-collector/http/registry"
	"github.com/benedict-erwin/insight-collector/http/v1/handler"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
)

func init() {
	// Register example router for v1
	registry.Register("v1", func(g *echo.Group) {
		logger.Info().Msg("Setting up /v1/example routes")
		g.POST("/example", handler.ExamplePost)
		g.GET("/example/:id", handler.ExampleGetId)
		g.POST("/example/job", handler.ExampleJob)
	})
}
