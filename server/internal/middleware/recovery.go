package middleware

import (
	"net/http"
	"runtime/debug"

	"promthus/internal/logger"
	"promthus/internal/model"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered",
					zap.Any("panic", r),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
					zap.String("request_id", model.GetRequestID(c)),
					zap.String("stack", string(debug.Stack())),
				)
				model.Fail(c, http.StatusInternalServerError, model.CodeInternalError, "internal server error")
				c.Abort()
			}
		}()
		c.Next()
	}
}
