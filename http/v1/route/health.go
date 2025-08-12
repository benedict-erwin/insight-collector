package route

import (
	"github.com/labstack/echo/v4"
	"github.com/benedict-erwin/insight-collector/http/middleware"
	"github.com/benedict-erwin/insight-collector/http/registry"
	"github.com/benedict-erwin/insight-collector/http/v1/handler"
	"github.com/benedict-erwin/insight-collector/pkg/auth"
)

// init registers v1 health check routes with the registry
func init() {
	registry.Register("v1", func(g *echo.Group) {
		// public
		g.GET("/health/live", handler.HealthLive)   // Liveness probe
		g.GET("/health/ready", handler.HealthReady) // Readiness probe

		// JWT protected
		jwtProtected := g.Group("")
		jwtProtected.Use(middleware.JWTAuthMiddleware(auth.ActionRead + ":health"))
		jwtProtected.GET("/health/jwt", handler.HealthDetailed) // JWT auth only
		
		// Signature protected
		sigProtected := g.Group("")
		sigProtected.Use(middleware.SignatureAuthMiddleware(auth.ActionRead + ":health"))
		sigProtected.GET("/health/signature", handler.HealthDetailed) // Signature auth only
		
		// Multi-auth (JWT or Signature)
		multiProtected := g.Group("")
		multiProtected.Use(middleware.MultiAuthMiddleware(auth.ActionRead + ":health"))
		multiProtected.GET("/health", handler.HealthDetailed) // Either JWT or Signature
	})
}
