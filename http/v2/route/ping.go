package route

import (
	"github.com/labstack/echo/v4"
	"github.com/benedict-erwin/insight-collector/http/middleware"
	"github.com/benedict-erwin/insight-collector/http/registry"
	"github.com/benedict-erwin/insight-collector/http/v1/handler"
	"github.com/benedict-erwin/insight-collector/pkg/auth"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
)

// init registers v2 ping routes with the registry
func init() {
	registry.Register("v2", func(g *echo.Group) {
		logger.Info().Msg("Setting up /v2/ping routes")
		
		// Multi-auth routes (supports both JWT and Signature)
		protected := g.Group("")
		protected.Use(middleware.MultiAuthMiddleware(auth.ActionRead + ":ping"))
		protected.GET("/ping", handler.Ping)
		protected.POST("/ping", handler.PingPost)
	})
}
