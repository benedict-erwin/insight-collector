package middleware

import (
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/benedict-erwin/insight-collector/config"
	"github.com/benedict-erwin/insight-collector/internal/constants"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
	"github.com/benedict-erwin/insight-collector/pkg/response"
)

// MultiAuthMiddleware creates middleware that supports both JWT and Signature authentication
func MultiAuthMiddleware(requiredPermission string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Setup logger scope
			log := logger.WithScope("MultiAuthMiddleware")

			// Check if auth is enabled
			authConfig := config.Get().Auth
			if !authConfig.Enabled {
				log.Debug().
					Str("path", c.Request().URL.Path).
					Str("method", c.Request().Method).
					Msg("Auth disabled, skipping multi authentication")
				return next(c)
			}
			// Check for JWT authentication (Bearer token)
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				log.Debug().
					Str("path", c.Request().URL.Path).
					Str("method", c.Request().Method).
					Msg("Using JWT authentication")
				return JWTAuthMiddleware(requiredPermission)(next)(c)
			}

			// Check for Signature authentication (X-Signature header)
			signature := c.Request().Header.Get("X-Signature")
			if signature != "" {
				log.Debug().
					Str("path", c.Request().URL.Path).
					Str("method", c.Request().Method).
					Msg("Using Signature authentication")
				return SignatureAuthMiddleware(requiredPermission)(next)(c)
			}

			// No authentication method found
			log.Warn().
				Str("path", c.Request().URL.Path).
				Str("method", c.Request().Method).
				Msg("No authentication method provided")
			return response.FailWithCode(c, constants.CodeMissingAuth)
		}
	}
}
