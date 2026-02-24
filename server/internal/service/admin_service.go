package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"promthus/internal/crypto"
	"promthus/internal/kms"
	"promthus/internal/logger"
	"promthus/internal/model"
	"promthus/internal/repository"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AdminService struct {
	sessionStore repository.SessionStore
}

func NewAdminService(ss repository.SessionStore) *AdminService {
	return &AdminService{sessionStore: ss}
}

// ==================== User Management ====================

type CreateUserRequest struct {
	Phone      string `json:"phone" binding:"required"`
	Name       string `json:"name" binding:"required,max=50"`
	Department string `json:"department"`
	Role       string `json:"role" binding:"required,oneof=user admin"`
}

type UpdateUserRequest struct {
	Name       *string `json:"name" binding:"omitempty,max=50"`
	Department *string `json:"department"`
	Role       *string `json:"role" binding:"omitempty,oneof=user admin"`
	Status     *int16  `json:"status" binding:"omitempty,oneof=0 1"`
}

func (s *AdminService) CreateUser(req *CreateUserRequest, operatorID int64) (*model.User, string, int, string) {
	logger.Info("create_user: start",
		zap.String("phone", req.Phone), zap.String("name", req.Name),
		zap.String("role", req.Role), zap.Int64("operator_id", operatorID))

	password, err := crypto.GenerateRandomPassword(16)
	if err != nil {
		logger.Error("create_user: generate password failed", zap.Error(err))
		return nil, "", model.CodeInternalError, "failed to generate password"
	}

	hash, err := crypto.HashPassword(password)
	if err != nil {
		logger.Error("create_user: hash password failed", zap.Error(err))
		return nil, "", model.CodeInternalError, "failed to hash password"
	}

	user := &model.User{
		UUID:         uuid.New(),
		Phone:        req.Phone,
		PasswordHash: hash,
		Name:         req.Name,
		Role:         req.Role,
		Status:       1,
	}
	if req.Department != "" {
		user.Department.String = req.Department
		user.Department.Valid = true
	}

	if err := repository.DB.Create(user).Error; err != nil {
		logger.Error("create_user: db insert failed", zap.Error(err), zap.String("phone", req.Phone))
		return nil, "", model.CodeInternalError, "failed to create user"
	}

	s.logOperation(operatorID, "create_user", "user", user.ID, nil, user)
	logger.Info("create_user: success",
		zap.Int64("user_id", user.ID), zap.String("uuid", user.UUID.String()),
		zap.String("role", req.Role), zap.Int64("operator_id", operatorID))

	return user, password, 0, ""
}

func (s *AdminService) UpdateUser(userUUID string, req *UpdateUserRequest, operatorID int64) (int, string) {
	logger.Info("update_user: start",
		zap.String("user_uuid", userUUID), zap.Int64("operator_id", operatorID))

	var user model.User
	if err := repository.DB.Where("uuid = ? AND deleted_at IS NULL", userUUID).First(&user).Error; err != nil {
		logger.Info("update_user: user not found", zap.String("user_uuid", userUUID))
		return model.CodeParamError, "user not found"
	}

	before := user

	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Department != nil {
		updates["department"] = *req.Department
	}
	if req.Role != nil {
		updates["role"] = *req.Role
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	updates["updated_at"] = time.Now()

	if err := repository.DB.Model(&user).Updates(updates).Error; err != nil {
		logger.Error("update_user: db update failed", zap.Error(err), zap.String("user_uuid", userUUID))
		return model.CodeInternalError, "update failed"
	}

	if req.Status != nil && *req.Status == 0 {
		_ = s.sessionStore.DeleteByUserID(user.ID)
		logger.Info("update_user: user disabled, all sessions cleared",
			zap.Int64("user_id", user.ID), zap.String("user_uuid", userUUID))
	}

	s.logOperation(operatorID, "update_user", "user", user.ID, before, updates)
	logger.Info("update_user: success",
		zap.Int64("user_id", user.ID), zap.String("user_uuid", userUUID),
		zap.Int64("operator_id", operatorID))

	return 0, ""
}

func (s *AdminService) ResetPassword(userUUID string, operatorID int64) (string, int, string) {
	var user model.User
	if err := repository.DB.Where("uuid = ? AND deleted_at IS NULL", userUUID).First(&user).Error; err != nil {
		logger.Info("reset_password: user not found", zap.String("user_uuid", userUUID), zap.Int64("operator_id", operatorID))
		return "", model.CodeParamError, "user not found"
	}

	password, err := crypto.GenerateRandomPassword(16)
	if err != nil {
		logger.Error("reset_password: generate password failed", zap.Error(err))
		return "", model.CodeInternalError, "failed to generate password"
	}
	hash, err := crypto.HashPassword(password)
	if err != nil {
		logger.Error("reset_password: hash password failed", zap.Error(err))
		return "", model.CodeInternalError, "failed to hash password"
	}

	if err := repository.DB.Model(&user).Updates(map[string]interface{}{
		"password_hash": hash,
		"updated_at":    time.Now(),
	}).Error; err != nil {
		logger.Error("reset_password: db update failed", zap.Error(err), zap.String("user_uuid", userUUID))
		return "", model.CodeInternalError, "failed to update password"
	}

	_ = s.sessionStore.DeleteByUserID(user.ID)
	s.logOperation(operatorID, "reset_password", "user", user.ID, nil, nil)

	logger.Info("reset_password success",
		zap.String("user_uuid", userUUID),
		zap.Int64("user_id", user.ID),
		zap.Int64("operator_id", operatorID),
	)

	return password, 0, ""
}

func (s *AdminService) ListUsers(page, pageSize int, role, status, search string) ([]model.User, int64) {
	query := repository.DB.Model(&model.User{}).Where("deleted_at IS NULL")

	if role != "" {
		query = query.Where("role = ?", role)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if search != "" {
		query = query.Where("name ILIKE ?", fmt.Sprintf("%%%s%%", search))
	}

	var total int64
	query.Count(&total)

	var users []model.User
	query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&users)

	for i := range users {
		users[i].PhoneMasked = crypto.MaskPhone(users[i].Phone)
	}

	return users, total
}

// ==================== Device Management ====================

type CreateDeviceRequest struct {
	DeviceID     string   `json:"device_id" binding:"required,max=32"`
	Name         string   `json:"name" binding:"required,max=100"`
	LocationText string   `json:"location_text" binding:"required"`
	Longitude    *float64 `json:"longitude"`
	Latitude     *float64 `json:"latitude"`
	PipelineTag  string   `json:"pipeline_tag"`
	RiskLevel    int16    `json:"risk_level" binding:"required,oneof=1 2 3"`
	DeviceKey    string   `json:"device_key" binding:"required"` // hex-encoded K_d
}

func (s *AdminService) CreateDevice(req *CreateDeviceRequest, operatorID int64) (*model.Device, int, string) {
	logger.Debug("create_device start", zap.String("device_id", req.DeviceID), zap.String("name", req.Name), zap.Int64("operator_id", operatorID))
	keyHex := strings.TrimSpace(strings.ReplaceAll(req.DeviceKey, " ", ""))
	if len(keyHex) != 32 {
		return nil, model.CodeParamError, "设备密钥须为 32 位十六进制（AES-128，如 0123456789abcdef0123456789abcdef）"
	}
	keyBytes, err := decodeHexKey(keyHex)
	if err != nil {
		return nil, model.CodeParamError, "设备密钥须为 32 位十六进制（仅含 0-9、a-f）"
	}
	defer clearBytes(keyBytes)

	encrypted, err := kms.Get().EncryptDeviceKey(keyBytes)
	if err != nil {
		logger.Error("KMS encrypt failed", zap.Error(err))
		return nil, model.CodeInternalError, "failed to encrypt device key"
	}

	device := &model.Device{
		DeviceID:     req.DeviceID,
		Name:         req.Name,
		LocationText: req.LocationText,
		Longitude:    req.Longitude,
		Latitude:     req.Latitude,
		RiskLevel:    req.RiskLevel,
		KeyEncrypted: encrypted,
		KeyVersion:   1,
		Status:       1,
	}
	if req.PipelineTag != "" {
		device.PipelineTag.String = req.PipelineTag
		device.PipelineTag.Valid = true
	}

	if err := repository.DB.Create(device).Error; err != nil {
		logger.Error("create_device db failed", zap.Error(err), zap.String("device_id", req.DeviceID))
		return nil, model.CodeInternalError, "failed to create device"
	}

	logger.Info("create_device success", zap.String("device_id", device.DeviceID), zap.Int64("id", device.ID))
	s.logOperation(operatorID, "create_device", "device", device.ID, nil, map[string]interface{}{
		"device_id": device.DeviceID, "name": device.Name,
	})

	return device, 0, ""
}

// ==================== Permission Management ====================

type GrantPermissionRequest struct {
	UserID     int64      `json:"user_id" binding:"required"`
	DeviceID   string     `json:"device_id" binding:"required,max=32"` // 业务编号，与 devices_lock.device_id 一致
	DeviceType string     `json:"device_type"`                         // 可选，默认 lock
	ValidFrom  time.Time  `json:"valid_from" binding:"required"`
	ValidUntil *time.Time `json:"valid_until"`
}

type BatchGrantRequest struct {
	Permissions []GrantPermissionRequest `json:"permissions" binding:"required,max=100,dive"`
}

func (s *AdminService) GrantPermission(req *GrantPermissionRequest, operatorID int64) (int, string) {
	deviceType := req.DeviceType
	if deviceType == "" {
		deviceType = model.DeviceTypeLock
	}
	logger.Debug("grant_permission start",
		zap.Int64("user_id", req.UserID),
		zap.String("device_id", req.DeviceID),
		zap.String("device_type", deviceType),
		zap.Int64("operator_id", operatorID),
		zap.Time("valid_from", req.ValidFrom),
	)
	var existing model.Permission
	err := repository.DB.Where("user_id = ? AND device_type = ? AND device_id = ? AND status = 1", req.UserID, deviceType, req.DeviceID).First(&existing).Error
	if err == nil {
		logger.Debug("grant_permission found existing, updating valid_until", zap.Int64("perm_id", existing.ID))
		if err := repository.DB.Model(&existing).Update("valid_until", req.ValidUntil).Error; err != nil {
			logger.Error("grant_permission update failed", zap.Error(err), zap.Int64("user_id", req.UserID), zap.String("device_id", req.DeviceID))
			return model.CodeInternalError, "更新授权失败"
		}
		s.logOperation(operatorID, "grant_permission", "permission", existing.ID, nil, req)
	} else {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Error("grant_permission lookup failed", zap.Error(err), zap.Int64("user_id", req.UserID), zap.String("device_id", req.DeviceID))
			return model.CodeInternalError, "查询授权失败"
		}
		var userCnt int64
		if repository.DB.Model(&model.User{}).Where("id = ? AND deleted_at IS NULL", req.UserID).Count(&userCnt).Error != nil || userCnt == 0 {
			logger.Info("grant_permission 400: 用户不存在", zap.Int64("user_id", req.UserID), zap.String("device_id", req.DeviceID))
			return model.CodeParamError, "用户不存在，请填写用户管理中的用户 ID（数字）"
		}
		// 设备存在性：按类型查对应表，当前仅 lock -> app.devices_lock
		if deviceType == model.DeviceTypeLock {
			var deviceCnt int64
			if repository.DB.Model(&model.Device{}).Where("device_id = ? AND deleted_at IS NULL", req.DeviceID).Count(&deviceCnt).Error != nil || deviceCnt == 0 {
				logger.Info("grant_permission 400: 设备不存在", zap.Int64("user_id", req.UserID), zap.String("device_id", req.DeviceID))
				return model.CodeParamError, "设备不存在，请填写锁具管理中的设备编号（device_id）"
			}
		}
		perm := &model.Permission{
			UserID:     req.UserID,
			DeviceType: deviceType,
			DeviceID:   req.DeviceID,
			GrantedBy:  operatorID,
			ValidFrom:  req.ValidFrom,
			ValidUntil: req.ValidUntil,
			Status:     1,
		}
		logger.Debug("grant_permission creating new permission")
		if err := repository.DB.Create(perm).Error; err != nil {
			logger.Error("grant_permission create failed", zap.Error(err), zap.Int64("user_id", req.UserID), zap.String("device_id", req.DeviceID))
			return model.CodeInternalError, "创建授权失败"
		}
		s.logOperation(operatorID, "grant_permission", "permission", perm.ID, nil, req)
	}
	logger.Info("grant_permission success", zap.Int64("user_id", req.UserID), zap.String("device_id", req.DeviceID))
	return 0, ""
}

func (s *AdminService) BatchGrantPermissions(req *BatchGrantRequest, operatorID int64) (int, string) {
	for _, p := range req.Permissions {
		if code, msg := s.GrantPermission(&p, operatorID); code != 0 {
			return code, msg
		}
	}
	return 0, ""
}

func (s *AdminService) RevokePermission(permID int64, operatorID int64) (int, string) {
	logger.Info("revoke_permission: start",
		zap.Int64("perm_id", permID), zap.Int64("operator_id", operatorID))

	result := repository.DB.Model(&model.Permission{}).
		Where("id = ? AND status = 1", permID).
		Updates(map[string]interface{}{
			"status":     0,
			"revoked_by": operatorID,
			"revoked_at": time.Now(),
		})

	if result.RowsAffected == 0 {
		logger.Info("revoke_permission: not found or already revoked", zap.Int64("perm_id", permID))
		return model.CodeParamError, "permission not found or already revoked"
	}

	s.logOperation(operatorID, "revoke_permission", "permission", permID, nil, nil)
	logger.Info("revoke_permission: success",
		zap.Int64("perm_id", permID), zap.Int64("operator_id", operatorID))
	return 0, ""
}

// ==================== Alert Management ====================

type HandleAlertRequest struct {
	HandleNote   string `json:"handle_note" binding:"required"`
	UnlockDevice bool   `json:"unlock_device"`
}

func (s *AdminService) HandleAlert(alertID int64, req *HandleAlertRequest, operatorID int64) (int, string) {
	logger.Info("handle_alert: start",
		zap.Int64("alert_id", alertID), zap.Bool("unlock_device", req.UnlockDevice),
		zap.Int64("operator_id", operatorID))

	var alert model.Alert
	if err := repository.DB.Where("id = ? AND status = 0", alertID).First(&alert).Error; err != nil {
		logger.Info("handle_alert: not found or already handled", zap.Int64("alert_id", alertID))
		return model.CodeParamError, "alert not found or already handled"
	}

	err := repository.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&alert).Updates(map[string]interface{}{
			"status":      1,
			"handled_by":  operatorID,
			"handle_note": req.HandleNote,
			"handled_at":  time.Now(),
		}).Error; err != nil {
			return err
		}

		if req.UnlockDevice && alert.AlertType == "consecutive_fail" && alert.DeviceType == model.DeviceTypeLock {
			logger.Info("handle_alert: unlocking device",
				zap.String("device_id", alert.DeviceID), zap.String("alert_type", alert.AlertType))
			return tx.Model(&model.Device{}).
				Where("device_id = ? AND status = 2", alert.DeviceID).
				Update("status", 1).Error
		}
		return nil
	})

	if err != nil {
		logger.Error("handle_alert: transaction failed", zap.Error(err), zap.Int64("alert_id", alertID))
		return model.CodeInternalError, "failed to handle alert"
	}

	s.logOperation(operatorID, "handle_alert", "alert", alertID, nil, req)
	logger.Info("handle_alert: success",
		zap.Int64("alert_id", alertID), zap.String("device_id", alert.DeviceID),
		zap.Int64("operator_id", operatorID))
	return 0, ""
}

func (s *AdminService) ListAlerts(status *int16, deviceID string, severity *int16, page, pageSize int) ([]model.Alert, int64) {
	query := repository.DB.Model(&model.Alert{})

	if status != nil {
		query = query.Where("status = ?", *status)
	}
	if deviceID != "" {
		query = query.Where("device_id = ?", deviceID)
	}
	if severity != nil {
		query = query.Where("severity = ?", *severity)
	}

	var total int64
	query.Count(&total)

	var alerts []model.Alert
	query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&alerts)

	return alerts, total
}

// ==================== Dashboard ====================

type DashboardData struct {
	TotalUsers      int64          `json:"total_users"`
	TotalDevices    int64          `json:"total_devices"`
	ActiveSessions  int64          `json:"active_sessions"`
	PendingAlerts   int64          `json:"pending_alerts"`
	RecentAlerts    []model.Alert  `json:"recent_alerts"`
	DevicesByStatus map[string]int64 `json:"devices_by_status"`
}

func (s *AdminService) GetDashboard() (*DashboardData, error) {
	data := &DashboardData{
		DevicesByStatus: make(map[string]int64),
	}

	repository.DB.Model(&model.User{}).Where("deleted_at IS NULL AND status = 1").Count(&data.TotalUsers)
	repository.DB.Model(&model.Device{}).Where("deleted_at IS NULL").Count(&data.TotalDevices)

	var err error
	data.ActiveSessions, err = s.sessionStore.CountActive()
	if err != nil {
		return nil, err
	}

	repository.DB.Model(&model.Alert{}).Where("status = 0").Count(&data.PendingAlerts)

	repository.DB.Model(&model.Alert{}).
		Where("status = 0").
		Order("created_at DESC").
		Limit(10).
		Find(&data.RecentAlerts)

	var statusCounts []struct {
		Status int16
		Count  int64
	}
	repository.DB.Model(&model.Device{}).
		Select("status, count(*) as count").
		Where("deleted_at IS NULL").
		Group("status").
		Scan(&statusCounts)

	statusNames := map[int16]string{0: "disabled", 1: "normal", 2: "alert_locked"}
	for _, sc := range statusCounts {
		name := statusNames[sc.Status]
		if name == "" {
			name = fmt.Sprintf("unknown_%d", sc.Status)
		}
		data.DevicesByStatus[name] = sc.Count
	}

	return data, nil
}

// ==================== Audit Logs ====================

func (s *AdminService) ListAuditLogs(userID *int64, deviceID, action string, startTime, endTime *time.Time, cursor string, limit int) (*model.PagedData, error) {
	query := repository.DB.Model(&model.AuditLog{})

	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}
	if deviceID != "" {
		query = query.Where("device_id = ?", deviceID)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if startTime != nil {
		query = query.Where("occurred_at >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("occurred_at <= ?", *endTime)
	}
	if cursor != "" {
		query = query.Where("occurred_at < ?", cursor)
	}

	var logs []model.AuditLog
	err := query.Order("occurred_at DESC").
		Limit(limit + 1).
		Find(&logs).Error
	if err != nil {
		return nil, err
	}

	hasMore := len(logs) > limit
	if hasMore {
		logs = logs[:limit]
	}

	nextCursor := ""
	if hasMore && len(logs) > 0 {
		nextCursor = logs[len(logs)-1].OccurredAt.Format(time.RFC3339Nano)
	}

	return &model.PagedData{
		Items:      logs,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

// ==================== Operation Logs ====================

func (s *AdminService) ListPermissions(userID *int64, deviceID *string, status *int16, page, pageSize int) ([]model.Permission, int64) {
	query := repository.DB.Model(&model.Permission{}).Preload("User")

	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}
	if deviceID != nil && *deviceID != "" {
		query = query.Where("device_id = ?", *deviceID)
	}
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	var total int64
	query.Count(&total)

	var perms []model.Permission
	query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&perms)

	return perms, total
}

// ==================== Helpers ====================

func (s *AdminService) logOperation(operatorID int64, action, targetType string, targetID int64, before, after interface{}) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("logOperation panic", zap.Any("panic", r), zap.String("action", action))
		}
	}()
	log := &model.OperationLog{
		OperatorID: operatorID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
	}
	if before != nil {
		log.BeforeSnapshot = toJSON(before)
	}
	if after != nil {
		log.AfterSnapshot = toJSON(after)
	}

	if err := repository.DB.Create(log).Error; err != nil {
		logger.Error("failed to log operation", zap.Error(err), zap.String("action", action))
	}
}

func toJSON(v interface{}) model.JSON {
	switch val := v.(type) {
	case model.JSON:
		return val
	case map[string]interface{}:
		return model.JSON(val)
	default:
		return model.JSON{"value": v}
	}
}

func decodeHexKey(hexStr string) ([]byte, error) {
	return decodeHex(hexStr)
}

func decodeHex(hexStr string) ([]byte, error) {
	b, err := hexDecode(hexStr)
	if err != nil {
		return nil, fmt.Errorf("invalid hex: %w", err)
	}
	return b, nil
}

func hexDecode(s string) ([]byte, error) {
	return hexDecodeImpl(s)
}

var hexDecodeImpl = func(s string) ([]byte, error) {
	b := make([]byte, len(s)/2)
	for i := 0; i < len(s); i += 2 {
		if i+1 >= len(s) {
			return nil, fmt.Errorf("odd hex length")
		}
		hi := hexCharToByte(s[i])
		lo := hexCharToByte(s[i+1])
		if hi == 0xFF || lo == 0xFF {
			return nil, fmt.Errorf("invalid hex char")
		}
		b[i/2] = (hi << 4) | lo
	}
	return b, nil
}

func hexCharToByte(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	default:
		return 0xFF
	}
}
