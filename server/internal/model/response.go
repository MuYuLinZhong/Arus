package model

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Response is the unified API response envelope.
type Response struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
	RequestID string      `json:"request_id"`
	Timestamp int64       `json:"timestamp"`
}

// PagedData wraps cursor-paginated list results.
type PagedData struct {
	Items      interface{} `json:"items"`
	NextCursor string      `json:"next_cursor"`
	HasMore    bool        `json:"has_more"`
}

func OK(c *gin.Context, data interface{}) {
	// 返回成功响应,包含请求ID,时间戳,数据;
	c.JSON(http.StatusOK, Response{
		Code:      0,
		Message:   "success",
		Data:      data,
		RequestID: GetRequestID(c),
		Timestamp: time.Now().UnixMilli(),
	})
}

func Fail(c *gin.Context, httpStatus int, bizCode int, message string) {
	c.JSON(httpStatus, Response{
		Code:      bizCode,
		Message:   message,
		Data:      nil,
		RequestID: GetRequestID(c),
		Timestamp: time.Now().UnixMilli(),
	})
}

func GetRequestID(c *gin.Context) string {
	if v, ok := c.Get("request_id"); ok {
		return v.(string)
	}
	return ""
}

// Business error code constants
const (
	// 1xxx - Authentication
	CodeAuthFailed      = 1001
	CodeAccountDisabled = 1002
	CodeSessionExpired  = 1003

	// 2xxx - Authorization
	CodeNoPermission = 2001
	CodeForbidden    = 2002

	// 3xxx - Lock operations
	CodeDeviceNotFound    = 3001
	CodeDeviceUnavailable = 3002
	CodeTooManyRequests   = 3003

	// 4xxx - Validation
	CodeParamError     = 4001
	CodeRequestExpired = 4002

	// 5xxx - Internal
	CodeInternalError = 5001
)
