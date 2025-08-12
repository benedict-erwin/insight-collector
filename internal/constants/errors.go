package constants

// Error Code Categories
// Format: XYZABC where:
// X = Category (1-9)
// YZ = Subcategory (00-99) 
// ABC = Specific error (000-999)

const (
	// SUCCESS CODES (0xxxx)
	CodeSuccess = 0

	// CLIENT ERROR CODES (4xxxx)
	// 400 Bad Request (40xxx)
	CodeBadRequest            = 40000 // Generic bad request
	CodeInvalidJSON           = 40001 // Invalid JSON payload
	CodeValidationFailed      = 40002 // Validation failed
	CodeMissingParameter      = 40003 // Required parameter missing
	CodeInvalidParameter      = 40004 // Invalid parameter value
	CodeInvalidFormat         = 40005 // Invalid format (email, etc)

	// 401 Unauthorized (41xxx)
	CodeUnauthorized          = 41000 // Generic unauthorized
	CodeMissingAuth           = 41001 // Missing authentication
	CodeInvalidToken          = 41002 // Invalid JWT token
	CodeExpiredToken          = 41003 // Expired JWT token
	CodeInvalidSignature      = 41004 // Invalid signature
	CodeExpiredSignature      = 41005 // Expired signature (timestamp)
	CodeInvalidClientID       = 41006 // Invalid client ID
	CodeInactiveClient        = 41007 // Client is inactive
	CodeNonceReplay           = 41008 // Nonce replay attack detected

	// 403 Forbidden (43xxx)
	CodeForbidden             = 43000 // Generic forbidden
	CodeInsufficientPerms     = 43001 // Insufficient permissions
	CodeResourceForbidden     = 43002 // Resource access forbidden

	// 404 Not Found (44xxx)
	CodeNotFound              = 44000 // Generic not found
	CodeResourceNotFound      = 44001 // Specific resource not found
	CodeEndpointNotFound      = 44002 // Endpoint not found

	// 409 Conflict (49xxx)
	CodeConflict              = 49000 // Generic conflict
	CodeDuplicateResource     = 49001 // Duplicate resource
	CodeDuplicateJob          = 49002 // Duplicate job

	// 422 Unprocessable Entity (42xxx)
	CodeUnprocessable         = 42000 // Generic unprocessable
	CodeBusinessLogicError    = 42001 // Business logic error
	CodeDependencyFailed      = 42002 // External dependency failed

	// 429 Too Many Requests (42xxx)
	CodeRateLimit             = 42900 // Rate limit exceeded

	// SERVER ERROR CODES (5xxxx)
	// 500 Internal Server Error (50xxx)
	CodeInternalError         = 50000 // Generic internal error
	CodeDatabaseError         = 50001 // Database error
	CodeInfluxDBError         = 50002 // InfluxDB error
	CodeRedisError            = 50003 // Redis error
	CodeJobProcessingError    = 50004 // Job processing error
	CodeConfigurationError    = 50005 // Configuration error
	CodeFileSystemError       = 50006 // File system error

	// 502 Bad Gateway (52xxx)
	CodeBadGateway            = 52000 // Generic bad gateway
	CodeUpstreamError         = 52001 // Upstream service error
	CodeExternalAPIError      = 52002 // External API error

	// 503 Service Unavailable (53xxx)
	CodeServiceUnavailable    = 53000 // Generic service unavailable
	CodeDatabaseUnavailable   = 53001 // Database unavailable
	CodeRedisUnavailable      = 53002 // Redis unavailable
	CodeMaintenanceMode       = 53003 // Maintenance mode

	// 504 Gateway Timeout (54xxx)
	CodeGatewayTimeout        = 54000 // Generic gateway timeout
	CodeUpstreamTimeout       = 54001 // Upstream timeout
	CodeDatabaseTimeout       = 54002 // Database timeout
)

// Error Code Messages - for consistent error messaging
var ErrorMessages = map[int]string{
	CodeSuccess: "Success",

	// Client Errors (4xxxx)
	CodeBadRequest:            "Bad request",
	CodeInvalidJSON:           "Invalid JSON payload",
	CodeValidationFailed:      "Validation failed",
	CodeMissingParameter:      "Required parameter missing",
	CodeInvalidParameter:      "Invalid parameter value",
	CodeInvalidFormat:         "Invalid format",

	CodeUnauthorized:          "Unauthorized",
	CodeMissingAuth:           "Authentication required: provide either Bearer token or X-Signature",
	CodeInvalidToken:          "Invalid JWT token",
	CodeExpiredToken:          "Token has expired",
	CodeInvalidSignature:      "Invalid signature",
	CodeExpiredSignature:      "Signature has expired",
	CodeInvalidClientID:       "Invalid client ID",
	CodeInactiveClient:        "Client is inactive",
	CodeNonceReplay:           "Nonce replay attack detected",

	CodeForbidden:             "Forbidden",
	CodeInsufficientPerms:     "Insufficient permissions",
	CodeResourceForbidden:     "Resource access forbidden",

	CodeNotFound:              "Not found",
	CodeResourceNotFound:      "Resource not found",
	CodeEndpointNotFound:      "Endpoint not found",

	CodeConflict:              "Conflict",
	CodeDuplicateResource:     "Duplicate resource",
	CodeDuplicateJob:          "Duplicate job - already in queue",

	CodeUnprocessable:         "Unprocessable entity",
	CodeBusinessLogicError:    "Business logic error",
	CodeDependencyFailed:      "External dependency failed",

	CodeRateLimit:             "Rate limit exceeded",

	// Server Errors (5xxxx)
	CodeInternalError:         "Internal server error",
	CodeDatabaseError:         "Database error",
	CodeInfluxDBError:         "InfluxDB error",
	CodeRedisError:            "Redis error",
	CodeJobProcessingError:    "Job processing error",
	CodeConfigurationError:    "Configuration error",
	CodeFileSystemError:       "File system error",

	CodeBadGateway:            "Bad gateway",
	CodeUpstreamError:         "Upstream service error",
	CodeExternalAPIError:      "External API error",

	CodeServiceUnavailable:    "Service unavailable",
	CodeDatabaseUnavailable:   "Database unavailable",
	CodeRedisUnavailable:      "Redis unavailable",
	CodeMaintenanceMode:       "Service under maintenance",

	CodeGatewayTimeout:        "Gateway timeout",
	CodeUpstreamTimeout:       "Upstream timeout",
	CodeDatabaseTimeout:       "Database timeout",
}

// GetErrorMessage returns the standard message for an error code
func GetErrorMessage(code int) string {
	if msg, exists := ErrorMessages[code]; exists {
		return msg
	}
	return "Unknown error"
}

// GetHTTPStatusFromCode returns the appropriate HTTP status code based on error code
func GetHTTPStatusFromCode(code int) int {
	switch {
	case code == 0:
		return 200
	case code >= 40000 && code < 41000:
		return 400
	case code >= 41000 && code < 42000:
		return 401
	case code >= 42000 && code < 43000:
		return 422
	case code >= 42900 && code < 43000:
		return 429
	case code >= 43000 && code < 44000:
		return 403
	case code >= 44000 && code < 45000:
		return 404
	case code >= 49000 && code < 50000:
		return 409
	case code >= 50000 && code < 51000:
		return 500
	case code >= 52000 && code < 53000:
		return 502
	case code >= 53000 && code < 54000:
		return 503
	case code >= 54000 && code < 55000:
		return 504
	default:
		return 500 // Default to internal server error
	}
}