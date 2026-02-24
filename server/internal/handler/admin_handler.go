package handler

import (
	"net/http"
	"strconv"
	"time"

	"promthus/internal/logger"
	"promthus/internal/model"
	"promthus/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// failWithLog 在返回 5xx 时写 Error 日志（含 request_id），便于排查
func failWithLog(c *gin.Context, code int, msg string) {
	status := httpStatusFromBizCode(code)
	if status >= 500 {
		logger.Error("admin handler error",
			zap.String("request_id", model.GetRequestID(c)),
			zap.String("path", c.FullPath()),
			zap.Int("http_status", status),
			zap.Int("biz_code", code),
			zap.String("message", msg),
		)
	}
	model.Fail(c, status, code, msg)
}

func failWithLogStatus(c *gin.Context, status, code int, msg string) {
	if status >= 500 {
		logger.Error("admin handler error",
			zap.String("request_id", model.GetRequestID(c)),
			zap.String("path", c.FullPath()),
			zap.Int("http_status", status),
			zap.Int("biz_code", code),
			zap.String("message", msg),
		)
	}
	model.Fail(c, status, code, msg)
}

type AdminHandler struct {
	svc *service.AdminService
}

func NewAdminHandler(svc *service.AdminService) *AdminHandler {
	return &AdminHandler{svc: svc}
}

// ==================== Users ====================

func (h *AdminHandler) ListUsers(c *gin.Context) {
	page, pageSize := parsePagination(c)
	role := c.Query("role")
	status := c.Query("status")
	search := c.Query("search")

	users, total := h.svc.ListUsers(page, pageSize, role, status, search)
	model.OK(c, gin.H{"items": users, "total": total})
}

func (h *AdminHandler) CreateUser(c *gin.Context) {
	var req service.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		model.Fail(c, http.StatusBadRequest, model.CodeParamError, "invalid request: "+err.Error())
		return
	}

	operatorID := c.GetInt64("user_id")
	user, _, code, msg := h.svc.CreateUser(&req, operatorID)
	if code != 0 {
		failWithLogStatus(c, http.StatusInternalServerError, code, msg)
		return
	}

	model.OK(c, user)
}

func (h *AdminHandler) UpdateUser(c *gin.Context) {
	userUUID := c.Param("uuid")
	var req service.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		model.Fail(c, http.StatusBadRequest, model.CodeParamError, "invalid request: "+err.Error())
		return
	}

	operatorID := c.GetInt64("user_id")
	code, msg := h.svc.UpdateUser(userUUID, &req, operatorID)
	if code != 0 {
		failWithLog(c, code, msg)
		return
	}

	model.OK(c, nil)
}

func (h *AdminHandler) ResetPassword(c *gin.Context) {
	userUUID := c.Param("uuid")
	operatorID := c.GetInt64("user_id")

	newPassword, code, msg := h.svc.ResetPassword(userUUID, operatorID)
	if code != 0 {
		failWithLog(c, code, msg)
		return
	}

	model.OK(c, gin.H{"new_password": newPassword})
}

// ==================== Devices ====================

func (h *AdminHandler) ListDevices(c *gin.Context) {
	page, pageSize := parsePagination(c)
	var status *int16
	if s := c.Query("status"); s != "" {
		v, _ := strconv.ParseInt(s, 10, 16)
		sv := int16(v)
		status = &sv
	}
	pipelineTag := c.Query("pipeline_tag")
	search := c.Query("search")

	lockSvc := service.NewLockService(nil, nil)
	devices, total, err := lockSvc.GetDeviceList(page, pageSize, status, pipelineTag, search)
	if err != nil {
		failWithLogStatus(c, http.StatusInternalServerError, model.CodeInternalError, "failed to list devices")
		return
	}

	model.OK(c, gin.H{"items": devices, "total": total})
}

func (h *AdminHandler) CreateDevice(c *gin.Context) {
	var req service.CreateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		model.Fail(c, http.StatusBadRequest, model.CodeParamError, "invalid request: "+err.Error())
		return
	}

	operatorID := c.GetInt64("user_id")
	device, code, msg := h.svc.CreateDevice(&req, operatorID)
	if code != 0 {
		failWithLog(c, code, msg)
		return
	}

	model.OK(c, device)
}

// ==================== Permissions ====================

func (h *AdminHandler) GrantPermission(c *gin.Context) {
	var req service.GrantPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Info("grant_permission 400: 参数校验失败", zap.String("request_id", model.GetRequestID(c)), zap.Error(err))
		model.Fail(c, http.StatusBadRequest, model.CodeParamError, "invalid request: "+err.Error())
		return
	}

	operatorID := c.GetInt64("user_id")
	code, msg := h.svc.GrantPermission(&req, operatorID)
	if code != 0 {
		failWithLog(c, code, msg)
		return
	}

	model.OK(c, nil)
}

func (h *AdminHandler) BatchGrantPermissions(c *gin.Context) {
	var req service.BatchGrantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		model.Fail(c, http.StatusBadRequest, model.CodeParamError, "invalid request: "+err.Error())
		return
	}

	operatorID := c.GetInt64("user_id")
	code, msg := h.svc.BatchGrantPermissions(&req, operatorID)
	if code != 0 {
		failWithLog(c, code, msg)
		return
	}

	model.OK(c, nil)
}

func (h *AdminHandler) RevokePermission(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		model.Fail(c, http.StatusBadRequest, model.CodeParamError, "invalid permission id")
		return
	}

	operatorID := c.GetInt64("user_id")
	code, msg := h.svc.RevokePermission(id, operatorID)
	if code != 0 {
		failWithLog(c, code, msg)
		return
	}

	model.OK(c, nil)
}

func (h *AdminHandler) ListPermissions(c *gin.Context) {
	page, pageSize := parsePagination(c)

	var userID *int64
	if v := c.Query("user_id"); v != "" {
		id, _ := strconv.ParseInt(v, 10, 64)
		userID = &id
	}
	deviceID := c.Query("device_id")
	var deviceIDPtr *string
	if deviceID != "" {
		deviceIDPtr = &deviceID
	}
	var status *int16
	if s := c.Query("status"); s != "" {
		v, _ := strconv.ParseInt(s, 10, 16)
		sv := int16(v)
		status = &sv
	}

	perms, total := h.svc.ListPermissions(userID, deviceIDPtr, status, page, pageSize)
	model.OK(c, gin.H{"items": perms, "total": total})
}

// ==================== Alerts ====================

func (h *AdminHandler) ListAlerts(c *gin.Context) {
	page, pageSize := parsePagination(c)

	var status *int16
	if s := c.Query("status"); s != "" {
		v, _ := strconv.ParseInt(s, 10, 16)
		sv := int16(v)
		status = &sv
	}
	deviceID := c.Query("device_id")
	var severity *int16
	if s := c.Query("severity"); s != "" {
		v, _ := strconv.ParseInt(s, 10, 16)
		sv := int16(v)
		severity = &sv
	}

	alerts, total := h.svc.ListAlerts(status, deviceID, severity, page, pageSize)
	model.OK(c, gin.H{"items": alerts, "total": total})
}

func (h *AdminHandler) HandleAlert(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		model.Fail(c, http.StatusBadRequest, model.CodeParamError, "invalid alert id")
		return
	}

	var req service.HandleAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		model.Fail(c, http.StatusBadRequest, model.CodeParamError, "invalid request: "+err.Error())
		return
	}

	operatorID := c.GetInt64("user_id")
	code, msg := h.svc.HandleAlert(id, &req, operatorID)
	if code != 0 {
		failWithLog(c, code, msg)
		return
	}

	model.OK(c, nil)
}

// ==================== Audit Logs ====================

func (h *AdminHandler) ListAuditLogs(c *gin.Context) {
	limit := 20
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}
	cursor := c.Query("cursor")

	var userID *int64
	if v := c.Query("user_id"); v != "" {
		id, _ := strconv.ParseInt(v, 10, 64)
		userID = &id
	}
	deviceID := c.Query("device_id")
	action := c.Query("action")

	var startTime, endTime *time.Time
	if v := c.Query("start_time"); v != "" {
		t, _ := time.Parse(time.RFC3339, v)
		startTime = &t
	}
	if v := c.Query("end_time"); v != "" {
		t, _ := time.Parse(time.RFC3339, v)
		endTime = &t
	}

	data, err := h.svc.ListAuditLogs(userID, deviceID, action, startTime, endTime, cursor, limit)
	if err != nil {
		failWithLogStatus(c, http.StatusInternalServerError, model.CodeInternalError, "failed to query audit logs")
		return
	}

	model.OK(c, data)
}

// ==================== Dashboard ====================

func (h *AdminHandler) Dashboard(c *gin.Context) {
	data, err := h.svc.GetDashboard()
	if err != nil {
		failWithLogStatus(c, http.StatusInternalServerError, model.CodeInternalError, "failed to get dashboard data")
		return
	}

	model.OK(c, data)
}

// ==================== Helpers ====================

func parsePagination(c *gin.Context) (int, int) {
	page := 1
	pageSize := 20

	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= 100 {
			pageSize = v
		}
	}

	return page, pageSize
}
