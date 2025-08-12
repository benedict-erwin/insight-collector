package middleware

import (
	"context"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/benedict-erwin/insight-collector/config"
	"github.com/benedict-erwin/insight-collector/internal/constants"
	"github.com/benedict-erwin/insight-collector/pkg/auth"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
	"github.com/benedict-erwin/insight-collector/pkg/response"
)

type contextKey string

const (
	ClientIDKey    contextKey = "client_id"
	ClientNameKey  contextKey = "client_name"
	PermissionsKey contextKey = "permissions"
)

// JWTAuthMiddleware creates JWT authentication middleware with required permission
func JWTAuthMiddleware(requiredPermission string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Setup logger scope
			log := logger.WithScope("JWTAuthMiddleware")

			// Check if auth is enabled
			authConfig := config.Get().Auth
			if !authConfig.Enabled {
				log.Debug().
					Str("path", c.Request().URL.Path).
					Str("method", c.Request().Method).
					Msg("Auth disabled, skipping JWT authentication")
				return next(c)
			}
			// Extract Bearer token from Authorization header
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				log.Warn().
					Str("path", c.Request().URL.Path).
					Str("method", c.Request().Method).
					Msg("Missing Authorization header")
				return response.FailWithCode(c, constants.CodeMissingAuth)
			}

			// Check Bearer prefix
			if !strings.HasPrefix(authHeader, "Bearer ") {
				log.Warn().
					Str("path", c.Request().URL.Path).
					Str("method", c.Request().Method).
					Msg("Invalid Authorization header format")
				return response.FailWithCode(c, constants.CodeInvalidToken)
			}

			// Extract token
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == "" {
				log.Warn().
					Str("path", c.Request().URL.Path).
					Str("method", c.Request().Method).
					Msg("Empty bearer token")
				return response.FailWithCode(c, constants.CodeInvalidToken)
			}

			// Verify JWT token
			claims, err := auth.VerifyJWT(tokenString)
			if err != nil {
				log.Warn().
					Err(err).
					Str("path", c.Request().URL.Path).
					Str("method", c.Request().Method).
					Msg("JWT verification failed")
				return response.FailWithCode(c, constants.CodeInvalidToken)
			}

			// Get client config from server (JWT verification already confirmed client exists)
			_, clientConfig, exists := auth.GetClientInfo(claims.ClientID)
			if !exists {
				log.Error().
					Str("client_id", claims.ClientID).
					Msg("Client config not found after successful JWT verification")
				return response.FailWithCode(c, constants.CodeInvalidToken)
			}

			// Check required permission using config permissions
			if requiredPermission != "" && !auth.HasPermission(clientConfig.Permissions, requiredPermission) {
				log.Warn().
					Str("client_id", claims.ClientID).
					Str("client_name", clientConfig.ClientName).
					Str("required_permission", requiredPermission).
					Strs("user_permissions", clientConfig.Permissions).
					Str("path", c.Request().URL.Path).
					Str("method", c.Request().Method).
					Msg("Insufficient permissions")
				return response.FailWithCode(c, constants.CodeInsufficientPerms)
			}

			// Set client context for handlers (using config data, not JWT claims)
			ctx := context.WithValue(c.Request().Context(), ClientIDKey, claims.ClientID)
			ctx = context.WithValue(ctx, ClientNameKey, clientConfig.ClientName)
			ctx = context.WithValue(ctx, PermissionsKey, clientConfig.Permissions)
			c.SetRequest(c.Request().WithContext(ctx))

			log.Info().
				Str("client_id", claims.ClientID).
				Str("client_name", clientConfig.ClientName).
				Str("required_permission", requiredPermission).
				Str("path", c.Request().URL.Path).
				Str("method", c.Request().Method).
				Msg("Authentication successful")

			return next(c)
		}
	}
}

// GetClientID extracts client ID from request context
func GetClientID(c echo.Context) string {
	if clientID, ok := c.Request().Context().Value(ClientIDKey).(string); ok {
		return clientID
	}
	return ""
}

// GetClientName extracts client name from request context
func GetClientName(c echo.Context) string {
	if clientName, ok := c.Request().Context().Value(ClientNameKey).(string); ok {
		return clientName
	}
	return ""
}

// GetPermissions extracts permissions from request context
func GetPermissions(c echo.Context) []string {
	if permissions, ok := c.Request().Context().Value(PermissionsKey).([]string); ok {
		return permissions
	}
	return []string{}
}
