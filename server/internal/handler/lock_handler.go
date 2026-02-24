package handler

import (
	"net/http"

	"promthus/internal/model"
	"promthus/internal/service"

	"github.com/gin-gonic/gin"
)

type LockHandler struct {
	svc *service.LockService
}

func NewLockHandler(svc *service.LockService) *LockHandler {
	return &LockHandler{svc: svc}
}

func (h *LockHandler) Challenge(c *gin.Context) {
	var req service.ChallengeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		model.Fail(c, http.StatusBadRequest, model.CodeParamError, "invalid request: "+err.Error())
		return
	}

	userID := c.GetInt64("user_id")
	resp, code, msg := h.svc.Challenge(&req, userID, c.ClientIP())
	if code != 0 {
		status := httpStatusFromBizCode(code)
		model.Fail(c, status, code, msg)
		return
	}

	model.OK(c, resp)
}

func (h *LockHandler) Report(c *gin.Context) {
	var req service.ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		model.Fail(c, http.StatusBadRequest, model.CodeParamError, "invalid request: "+err.Error())
		return
	}

	userID := c.GetInt64("user_id")
	code, msg := h.svc.Report(&req, userID, c.ClientIP())
	if code != 0 {
		model.Fail(c, httpStatusFromBizCode(code), code, msg)
		return
	}

	model.OK(c, nil)
}

func (h *LockHandler) GetDevices(c *gin.Context) {
	userID := c.GetInt64("user_id")
	devices, err := h.svc.GetAuthorizedDevices(userID)
	if err != nil {
		model.Fail(c, http.StatusInternalServerError, model.CodeInternalError, "failed to get devices")
		return
	}

	model.OK(c, devices)
}

func httpStatusFromBizCode(code int) int {
	switch {
	case code >= 1000 && code < 2000:
		return http.StatusUnauthorized
	case code >= 2000 && code < 3000:
		return http.StatusForbidden
	case code == model.CodeTooManyRequests:
		return http.StatusTooManyRequests
	case code >= 3000 && code < 4000:
		return http.StatusBadRequest
	case code >= 4000 && code < 5000:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
