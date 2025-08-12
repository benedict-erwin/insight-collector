package route

import (
	"github.com/labstack/echo/v4"
	"github.com/benedict-erwin/insight-collector/http/registry"
	"github.com/benedict-erwin/insight-collector/http/v1/handler"
)

func init() {
	// Register user activities routes for v1
	registry.Register("v1", func(g *echo.Group) {
		ua := g.Group("/callback-logs")
		ua.POST("/insert", handler.SaveCallbackLogs)
		ua.POST("/list", handler.ListCallbackLogs)
		ua.GET("/:id", handler.DetailCallbackLogs)
	})
}
