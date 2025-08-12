package middleware

import (
	"context"
	"io"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/benedict-erwin/insight-collector/config"
	"github.com/benedict-erwin/insight-collector/internal/constants"
	"github.com/benedict-erwin/insight-collector/pkg/auth"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
	"github.com/benedict-erwin/insight-collector/pkg/response"
)

// SignatureAuthMiddleware creates signature-based authentication middleware with required permission
func SignatureAuthMiddleware(requiredPermission string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Setup logger scope
			log := logger.WithScope("SignatureAuthMiddleware")

			// Check if auth is enabled
			authConfig := config.Get().Auth
			if !authConfig.Enabled {
				log.Debug().
					Str("path", c.Request().URL.Path).
					Str("method", c.Request().Method).
					Msg("Auth disabled, skipping signature authentication")
				return next(c)
			}
			// Extract signature components from headers
			clientID := c.Request().Header.Get("X-Client-ID")
			timestamp := c.Request().Header.Get("X-Timestamp")
			nonce := c.Request().Header.Get("X-Nonce") // Optional
			signature := c.Request().Header.Get("X-Signature")

			if clientID == "" {
				log.Warn().
					Str("path", c.Request().URL.Path).
					Str("method", c.Request().Method).
					Msg("Missing X-Client-ID header")
				return response.FailWithCode(c, constants.CodeUnauthorized)
			}

			if timestamp == "" {
				log.Warn().
					Str("path", c.Request().URL.Path).
					Str("method", c.Request().Method).
					Str("client_id", clientID).
					Msg("Missing X-Timestamp header")
				return response.FailWithCode(c, constants.CodeExpiredSignature)
			}

			if signature == "" {
				log.Warn().
					Str("path", c.Request().URL.Path).
					Str("method", c.Request().Method).
					Str("client_id", clientID).
					Msg("Missing X-Signature header")
				return response.FailWithCode(c, constants.CodeMissingAuth)
			}

			// Read request body for signature verification
			body := ""
			if c.Request().Body != nil {
				bodyBytes, err := io.ReadAll(c.Request().Body)
				if err != nil {
					log.Warn().
						Err(err).
						Str("client_id", clientID).
						Msg("Failed to read request body")
					return response.FailWithCode(c, constants.CodeBadRequest)
				}
				body = string(bodyBytes)

				// Restore request body for downstream handlers
				c.Request().Body = io.NopCloser(strings.NewReader(body))
			}

			// Verify signature
			clientConfig, err := auth.VerifySignature(
				clientID,
				timestamp,
				nonce,
				c.Request().Method,
				c.Request().URL.Path,
				body,
				signature,
			)
			if err != nil {
				log.Warn().
					Err(err).
					Str("path", c.Request().URL.Path).
					Str("method", c.Request().Method).
					Str("client_id", clientID).
					Msg("Signature verification failed")
				return response.FailWithCode(c, constants.CodeInvalidSignature)
			}

			// Check required permission using config permissions
			if requiredPermission != "" && !auth.HasPermission(clientConfig.Permissions, requiredPermission) {
				log.Warn().
					Str("client_id", clientID).
					Str("client_name", clientConfig.ClientName).
					Str("auth_type", clientConfig.AuthType).
					Str("required_permission", requiredPermission).
					Strs("user_permissions", clientConfig.Permissions).
					Str("path", c.Request().URL.Path).
					Str("method", c.Request().Method).
					Msg("Insufficient permissions")
				return response.FailWithCode(c, constants.CodeInsufficientPerms)
			}

			// Set client context for handlers (using config data)
			ctx := context.WithValue(c.Request().Context(), ClientIDKey, clientConfig.ClientID)
			ctx = context.WithValue(ctx, ClientNameKey, clientConfig.ClientName)
			ctx = context.WithValue(ctx, PermissionsKey, clientConfig.Permissions)
			c.SetRequest(c.Request().WithContext(ctx))

			log.Info().
				Str("client_id", clientID).
				Str("client_name", clientConfig.ClientName).
				Str("auth_type", clientConfig.AuthType).
				Str("required_permission", requiredPermission).
				Str("path", c.Request().URL.Path).
				Str("method", c.Request().Method).
				Msg("Signature authentication successful")

			return next(c)
		}
	}
}
