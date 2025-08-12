package route

import (
	"github.com/labstack/echo/v4"
	"github.com/benedict-erwin/insight-collector/http/registry"
	"github.com/benedict-erwin/insight-collector/http/v1/handler"
)

func init() {
	// Register user activities routes for v1
	registry.Register("v1", func(g *echo.Group) {
		ua := g.Group("/security-events")
		ua.POST("/insert", handler.SaveSecurityEvents)
		ua.POST("/list", handler.ListSecurityEvents)
		ua.GET("/:id", handler.DetailSecurityEvents)
	})
}
