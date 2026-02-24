package handler

import (
	"net/http"

	"promthus/internal/logger"
	"promthus/internal/middleware"
	"promthus/internal/model"
	"promthus/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		model.Fail(c, http.StatusBadRequest, model.CodeParamError, "invalid request: "+err.Error())
		return
	}

	// 调用服务层的登录方法,进行登录检查
	resp, code, msg := h.svc.Login(&req, c.Request.UserAgent(), c.ClientIP())
	if code != 0 {
		status := http.StatusUnauthorized
		if code == model.CodeAccountDisabled {
			status = http.StatusForbidden
		} else if code == model.CodeInternalError {
			status = http.StatusInternalServerError
		}
		if status >= 500 {
			logger.Error("auth login error", zap.String("request_id", model.GetRequestID(c)), zap.Int("code", code), zap.String("msg", msg))
		}
		model.Fail(c, status, code, msg)
		return
	}

	model.OK(c, resp)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	header := c.GetHeader("Authorization")
	tokenStr := header[7:]
	claims, err := middleware.ParseToken(tokenStr)
	if err != nil {
		model.Fail(c, http.StatusUnauthorized, model.CodeSessionExpired, "invalid token")
		return
	}

	if err := h.svc.Logout(claims.JTI); err != nil {
		logger.Error("logout failed", zap.String("request_id", model.GetRequestID(c)), zap.Error(err))
		model.Fail(c, http.StatusInternalServerError, model.CodeInternalError, "logout failed")
		return
	}

	model.OK(c, nil)
}
