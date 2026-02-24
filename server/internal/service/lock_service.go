package service

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	"promthus/internal/kms"
	"promthus/internal/logger"
	"promthus/internal/model"
	"promthus/internal/mq"
	"promthus/internal/repository"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type LockService struct {
	failStore repository.DeviceFailStore
	publisher *mq.Publisher
}

func NewLockService(fs repository.DeviceFailStore, pub *mq.Publisher) *LockService {
	return &LockService{failStore: fs, publisher: pub}
}

type ChallengeRequest struct {
	DeviceID   string `json:"device_id" binding:"required,max=32"`
	ChallengeC string `json:"challenge_c" binding:"required,len=16"`
	Timestamp  int64  `json:"timestamp" binding:"required"`
}

type ChallengeResponse struct {
	Response string `json:"response"`
}

type ReportRequest struct {
	DeviceID    string `json:"device_id" binding:"required,max=32"`
	Result      string `json:"result" binding:"required,oneof=success fail"`
	FailReason  string `json:"fail_reason"`
	OccurredAt  int64  `json:"occurred_at" binding:"required"`
	DeviceModel string `json:"device_model"`
}

func (s *LockService) Challenge(req *ChallengeRequest, userID int64, clientIP string) (*ChallengeResponse, int, string) {
	logger.Info("challenge: start",
		zap.Int64("user_id", userID),
		zap.String("device_id", req.DeviceID),
		zap.String("client_ip", clientIP),
	)

	if _, err := hex.DecodeString(req.ChallengeC); err != nil || len(req.ChallengeC) != 16 {
		logger.Info("challenge: rejected, invalid challenge_c",
			zap.Int64("user_id", userID), zap.String("device_id", req.DeviceID))
		return nil, model.CodeParamError, "challenge_c must be 16 hex characters (8 bytes)"
	}

	serverTime := time.Now().Unix()
	if math.Abs(float64(serverTime-req.Timestamp)) > 30 {
		logger.Info("challenge: rejected, timestamp drift",
			zap.Int64("user_id", userID), zap.String("device_id", req.DeviceID),
			zap.Int64("client_ts", req.Timestamp), zap.Int64("server_ts", serverTime))
		return nil, model.CodeRequestExpired, "request expired"
	}

	var device model.Device
	err := repository.DB.Where("device_id = ? AND deleted_at IS NULL", req.DeviceID).First(&device).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Info("challenge: rejected, device not found",
				zap.Int64("user_id", userID), zap.String("device_id", req.DeviceID))
			return nil, model.CodeDeviceNotFound, "device not found"
		}
		logger.Error("challenge: device query failed", zap.Error(err))
		return nil, model.CodeInternalError, "internal error"
	}

	if device.Status != 1 {
		logger.Info("challenge: rejected, device unavailable",
			zap.String("device_id", req.DeviceID), zap.Int16("device_status", device.Status))
		statusMsg := "device unavailable"
		if device.Status == 2 {
			statusMsg = "device locked due to security alert"
		}
		return nil, model.CodeDeviceUnavailable, statusMsg
	}

	var permCount int64
	repository.DB.Model(&model.Permission{}).
		Where("user_id = ? AND device_type = ? AND device_id = ? AND status = 1 AND valid_from <= ? AND (valid_until IS NULL OR valid_until > ?)",
			userID, model.DeviceTypeLock, device.DeviceID, time.Now(), time.Now()).
		Count(&permCount)
	if permCount == 0 {
		logger.Info("challenge: rejected, no permission",
			zap.Int64("user_id", userID), zap.String("device_id", req.DeviceID))
		return nil, model.CodeNoPermission, "no permission for this device"
	}

	kd, err := kms.Get().DecryptDeviceKey(device.KeyEncrypted)
	if err != nil {
		logger.Error("challenge: KMS decrypt failed", zap.Error(err), zap.String("device_id", req.DeviceID))
		return nil, model.CodeInternalError, "internal error"
	}
	defer clearBytes(kd)

	challengeBytes, _ := hex.DecodeString(req.ChallengeC)
	deviceIDBytes := []byte(req.DeviceID)
	userIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(userIDBytes, uint64(userID))
	tsBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(tsBytes, uint64(req.Timestamp))

	data := append(challengeBytes, deviceIDBytes...)
	data = append(data, userIDBytes...)
	data = append(data, tsBytes...)

	cmacResult, err := kms.Get().ComputeCMAC(kd, data)
	if err != nil {
		logger.Error("challenge: CMAC computation failed", zap.Error(err))
		return nil, model.CodeInternalError, "internal error"
	}

	logger.Info("challenge: success, response computed",
		zap.Int64("user_id", userID), zap.String("device_id", req.DeviceID))

	if s.publisher != nil {
		_ = s.publisher.PublishAudit(&mq.AuditMessage{
			UserID:     userID,
			DeviceID:   req.DeviceID,
			DeviceType: model.DeviceTypeLock,
			Action:     "challenge_request",
			ClientIP:   clientIP,
		})
	}

	go func() {
		repository.DB.Model(&model.Device{}).
			Where("device_id = ? AND deleted_at IS NULL", req.DeviceID).
			Update("last_active_at", time.Now())
	}()

	return &ChallengeResponse{
		Response: hex.EncodeToString(cmacResult),
	}, 0, ""
}

func (s *LockService) Report(req *ReportRequest, userID int64, clientIP string) (int, string) {
	logger.Info("report: received",
		zap.Int64("user_id", userID),
		zap.String("device_id", req.DeviceID),
		zap.String("result", req.Result),
		zap.String("client_ip", clientIP),
	)

	if req.Result == "fail" {
		count, err := s.failStore.Increment(model.DeviceTypeLock, req.DeviceID)
		if err != nil {
			logger.Error("report: fail count increment error", zap.Error(err))
		}

		logger.Info("report: unlock failed, fail_count incremented",
			zap.String("device_id", req.DeviceID), zap.Int("fail_count", count),
			zap.String("fail_reason", req.FailReason))

		if count >= 3 {
			logger.Warn("report: consecutive fail threshold reached, triggering alert",
				zap.String("device_id", req.DeviceID), zap.Int("fail_count", count))
			s.triggerAlertLock(req.DeviceID, userID, count)
		}
	} else {
		_ = s.failStore.Reset(model.DeviceTypeLock, req.DeviceID)
		repository.DB.Model(&model.Device{}).
			Where("device_id = ? AND deleted_at IS NULL", req.DeviceID).
			Update("last_active_at", time.Now())
		logger.Info("report: unlock success, fail_count reset",
			zap.String("device_id", req.DeviceID), zap.Int64("user_id", userID))
	}

	action := "unlock_success"
	resultCode := int16(0)
	if req.Result == "fail" {
		action = "unlock_fail"
		resultCode = 1
	}

	if s.publisher != nil {
		_ = s.publisher.PublishAudit(&mq.AuditMessage{
			UserID:      userID,
			DeviceID:    req.DeviceID,
			DeviceType:  model.DeviceTypeLock,
			Action:      action,
			ResultCode:  resultCode,
			ClientIP:    clientIP,
			DeviceModel: req.DeviceModel,
			Extra:       map[string]interface{}{"fail_reason": req.FailReason},
		})
	}

	return 0, ""
}

func (s *LockService) GetAuthorizedDevices(userID int64) ([]model.Device, error) {
	var devices []model.Device
	now := time.Now()
	err := repository.DB.Model(&model.Device{}).
		Joins("JOIN app.permissions ON app.permissions.device_type = ? AND app.permissions.device_id = app.devices_lock.device_id AND app.permissions.user_id = ? AND app.permissions.status = 1 AND app.permissions.valid_from <= ? AND (app.permissions.valid_until IS NULL OR app.permissions.valid_until > ?)",
			model.DeviceTypeLock, userID, now, now).
		Where("app.devices_lock.deleted_at IS NULL AND app.devices_lock.status != 0").
		Find(&devices).Error
	return devices, err
}

func (s *LockService) triggerAlertLock(deviceID string, userID int64, failCount int) {
	logger.Info("triggerAlertLock: locking device and creating alert",
		zap.String("device_id", deviceID), zap.Int64("user_id", userID), zap.Int("fail_count", failCount))

	err := repository.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Device{}).
			Where("device_id = ? AND deleted_at IS NULL", deviceID).
			Update("status", 2).Error; err != nil {
			return err
		}

		alert := &model.Alert{
			AlertType:  "consecutive_fail",
			DeviceType: model.DeviceTypeLock,
			DeviceID:   deviceID,
			UserID:     &userID,
			Severity:   3,
			Status:     0,
			Extra:      model.JSON{"fail_count": failCount},
		}
		if err := tx.Create(alert).Error; err != nil {
			return err
		}

		return tx.Exec("UPDATE app.device_fail_counts SET count = 0 WHERE device_type = ? AND device_id = ?", model.DeviceTypeLock, deviceID).Error
	})

	if err != nil {
		logger.Error("triggerAlertLock: transaction failed",
			zap.String("device_id", deviceID), zap.Error(err))
		return
	}

	logger.Warn("triggerAlertLock: device locked, alert created",
		zap.String("device_id", deviceID), zap.Int("fail_count", failCount))

	if s.publisher != nil {
		_ = s.publisher.PublishNotify(&mq.NotifyMessage{
			AlertType: "consecutive_fail",
			DeviceID:  deviceID,
			Severity:  3,
			Extra:     map[string]interface{}{"fail_count": failCount},
		})
	}
}

func clearBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func (s *LockService) GetDeviceList(page, pageSize int, status *int16, pipelineTag, search string) ([]model.Device, int64, error) {
	query := repository.DB.Model(&model.Device{}).Where("deleted_at IS NULL")

	if status != nil {
		query = query.Where("status = ?", *status)
	}
	if pipelineTag != "" {
		query = query.Where("pipeline_tag = ?", pipelineTag)
	}
	if search != "" {
		query = query.Where("device_id ILIKE ? OR name ILIKE ?",
			fmt.Sprintf("%%%s%%", search), fmt.Sprintf("%%%s%%", search))
	}

	var total int64
	query.Count(&total)

	var devices []model.Device
	err := query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&devices).Error

	return devices, total, err
}
