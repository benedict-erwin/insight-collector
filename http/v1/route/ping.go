package route

import (
	"github.com/labstack/echo/v4"
	"github.com/benedict-erwin/insight-collector/http/middleware"
	"github.com/benedict-erwin/insight-collector/http/registry"
	"github.com/benedict-erwin/insight-collector/http/v1/handler"
	"github.com/benedict-erwin/insight-collector/pkg/auth"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
)

// init registers v1 ping routes with the registry
func init() {
	registry.Register("v1", func(g *echo.Group) {
		logger.Info().Msg("Setting up /v1/ping routes")
		
		// Multi-auth routes (default - supports both JWT and Signature)
		protected := g.Group("")
		protected.Use(middleware.MultiAuthMiddleware(auth.ActionRead + ":ping"))
		protected.GET("/ping", handler.Ping)
		protected.POST("/ping", handler.PingPost)
		
		// JWT only routes (for specific JWT testing)
		jwtOnly := g.Group("")
		jwtOnly.Use(middleware.JWTAuthMiddleware(auth.ActionRead + ":ping"))
		jwtOnly.GET("/ping/jwt", handler.Ping)
		
		// Signature only routes (for specific signature testing)
		sigOnly := g.Group("")
		sigOnly.Use(middleware.SignatureAuthMiddleware(auth.ActionRead + ":ping"))
		sigOnly.GET("/ping/signature", handler.Ping)
	})
}
