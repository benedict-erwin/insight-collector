package response

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/benedict-erwin/insight-collector/internal/constants"
)

// Standard Response struct
type Response struct {
	Success   bool   `json:"success"`
	Code      int    `json:"code"`
	Data      any    `json:"data"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

// getReqId extracts request ID from Echo context
func getReqId(c echo.Context) string {
	return constants.GetRequestID(c)
}

// Success returns a successful response with data
func Success(c echo.Context, data any) error {
	return c.JSON(http.StatusOK, Response{
		Success:   true,
		Code:      0,
		Data:      data,
		Message:   "Successful",
		RequestID: getReqId(c),
	})
}

// Fail returns an error response with message
func Fail(c echo.Context, httpStatus int, code int, message string) error {
	return c.JSON(httpStatus, Response{
		Success:   false,
		Code:      code,
		Data:      nil,
		Message:   message,
		RequestID: getReqId(c),
	})
}

// General returns a customizable response
func General(c echo.Context, httpStatus int, code int, data any, message string) error {
	return c.JSON(httpStatus, Response{
		Success:   httpStatus < 400,
		Code:      code,
		Data:      data,
		Message:   message,
		RequestID: getReqId(c),
	})
}

// FailWithCode returns an error response using standardized error code
func FailWithCode(c echo.Context, code int) error {
	httpStatus := constants.GetHTTPStatusFromCode(code)
	message := constants.GetErrorMessage(code)
	return c.JSON(httpStatus, Response{
		Success:   false,
		Code:      code,
		Data:      nil,
		Message:   message,
		RequestID: getReqId(c),
	})
}

// FailWithCodeAndMessage returns an error response with custom message
func FailWithCodeAndMessage(c echo.Context, code int, customMessage string) error {
	httpStatus := constants.GetHTTPStatusFromCode(code)
	return c.JSON(httpStatus, Response{
		Success:   false,
		Code:      code,
		Data:      nil,
		Message:   customMessage,
		RequestID: getReqId(c),
	})
}
