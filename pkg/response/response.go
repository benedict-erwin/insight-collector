package response

import (
	"bytes"
	"net/http"
	"sync"

	"github.com/goccy/go-json"
	"github.com/labstack/echo/v4"
	"github.com/benedict-erwin/insight-collector/internal/constants"
)

// Buffer pool for high-performance JSON encoding
var bufferPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

// getBuffer gets a buffer from pool
func getBuffer() *bytes.Buffer {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// putBuffer returns buffer to pool (only if not too large)
func putBuffer(buf *bytes.Buffer) {
	// Prevent memory leak from oversized buffers (>64KB)
	const maxBufferSize = 64 * 1024
	if buf.Cap() < maxBufferSize {
		bufferPool.Put(buf)
	}
}

// fastJSON performs high-performance JSON serialization with buffer pooling
func fastJSON(c echo.Context, code int, obj interface{}) error {
	buf := getBuffer()
	defer putBuffer(buf)

	// High-performance JSON encoding
	if err := json.NewEncoder(buf).Encode(obj); err != nil {
		return err
	}

	// Set content type and send response
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
	c.Response().WriteHeader(code)
	_, err := c.Response().Write(buf.Bytes())
	return err
}

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
	return fastJSON(c, http.StatusOK, Response{
		Success:   true,
		Code:      0,
		Data:      data,
		Message:   "Successful",
		RequestID: getReqId(c),
	})
}

// Fail returns an error response with message
func Fail(c echo.Context, httpStatus int, code int, message string) error {
	return fastJSON(c, httpStatus, Response{
		Success:   false,
		Code:      code,
		Data:      nil,
		Message:   message,
		RequestID: getReqId(c),
	})
}

// General returns a customizable response
func General(c echo.Context, httpStatus int, code int, data any, message string) error {
	return fastJSON(c, httpStatus, Response{
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
	return fastJSON(c, httpStatus, Response{
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
	return fastJSON(c, httpStatus, Response{
		Success:   false,
		Code:      code,
		Data:      nil,
		Message:   customMessage,
		RequestID: getReqId(c),
	})
}
