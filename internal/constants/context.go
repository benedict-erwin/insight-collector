package constants

import "github.com/labstack/echo/v4"

const (
	// Internal usage
	RequestIDKey = "x-req-id"

	// Header keys (in order of preference)
	HeaderRequestID      = "X-Request-ID"     // Primary standard
	HeaderCorrelationID  = "X-Correlation-ID" // Alternative
	HeaderRequestIDShort = "Request-ID"       // Modern format
)

// GetRequestIDFromHeaders extracts request ID from multiple possible headers
func GetRequestIDFromHeaders(c echo.Context) string {
	// Priority order: X-Request-ID > X-Correlation-ID > Request-ID
	if xReqId := c.Request().Header.Get(HeaderRequestID); xReqId != "" {
		return xReqId
	}
	if xCorId := c.Request().Header.Get(HeaderCorrelationID); xCorId != "" {
		return xCorId
	}
	if reqId := c.Request().Header.Get(HeaderRequestIDShort); reqId != "" {
		return reqId
	}
	return ""
}

// GetRequestID extracts request ID from Echo context
func GetRequestID(c echo.Context) string {
	rid, ok := c.Get(RequestIDKey).(string)
	if !ok {
		return ""
	}
	return rid
}
