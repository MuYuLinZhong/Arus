// 本文件提供会话存储的接口与实现。
//
// SessionStore 接口（可类比 C++ 抽象基类）：
//   - 只约定一组方法签名，不包含实现；
//   - 任意类型只要实现了这些方法，即可当作 SessionStore 使用，无需显式声明“继承”；
//   - 业务层（如 AuthService）统一以 SessionStore 为入参类型，可传入 PostgresSessionStore、日后的 RedisSessionStore 等；
//   - 调用时通过接口的方法表多态到具体实现，类似基类指针/虚表分发。
//
// 这样便于替换实现（如从 DB 换 Redis）和做单元测试（MockSessionStore），而不改调用方代码。
package repository

import (
	"time"

	"promthus/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SessionStore abstracts session persistence. Implement RedisSessionStore later for scaling.
type SessionStore interface {
	Create(session *model.Session) error
	FindByJTI(jti uuid.UUID) (*model.Session, error)
	DeleteByJTI(jti uuid.UUID) error
	DeleteByUserID(userID int64) error
	CleanExpired() (int64, error)
	CountActive() (int64, error)
}

// 类似于定义一个类，实现 SessionStore 接口;
type PostgresSessionStore struct{}

func NewPostgresSessionStore() SessionStore {
	return &PostgresSessionStore{}
}

// 实现 SessionStore 接口的 Create 方法等一些列的方法;
func (s *PostgresSessionStore) Create(session *model.Session) error {
	return DB.Create(session).Error
}

func (s *PostgresSessionStore) FindByJTI(jti uuid.UUID) (*model.Session, error) {
	var session model.Session
	err := DB.Where("jti = ? AND expires_at > ?", jti, time.Now()).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *PostgresSessionStore) DeleteByJTI(jti uuid.UUID) error {
	return DB.Where("jti = ?", jti).Delete(&model.Session{}).Error
}

func (s *PostgresSessionStore) DeleteByUserID(userID int64) error {
	return DB.Where("user_id = ?", userID).Delete(&model.Session{}).Error
}

func (s *PostgresSessionStore) CleanExpired() (int64, error) {
	result := DB.Where("expires_at < ?", time.Now()).Delete(&model.Session{})
	return result.RowsAffected, result.Error
}

func (s *PostgresSessionStore) CountActive() (int64, error) {
	var count int64
	err := DB.Model(&model.Session{}).Where("expires_at > ?", time.Now()).Count(&count).Error
	return count, err
}

// RateLimitStore abstracts rate limiting persistence.
type RateLimitStore interface {
	Increment(key string, windowSecs int) (int, error)
}

// DeviceFailStore abstracts device failure counting. Key is (device_type, device_id).
type DeviceFailStore interface {
	Increment(deviceType, deviceID string) (int, error)
	Reset(deviceType, deviceID string) error
	Get(deviceType, deviceID string) (int, error)
}

type PostgresDeviceFailStore struct{}

func NewPostgresDeviceFailStore() DeviceFailStore {
	return &PostgresDeviceFailStore{}
}

func (s *PostgresDeviceFailStore) Increment(deviceType, deviceID string) (int, error) {
	var fc model.DeviceFailCount
	sql := `INSERT INTO app.device_fail_counts (device_type, device_id, count, last_fail_at, updated_at)
		VALUES (?, ?, 1, NOW(), NOW())
		ON CONFLICT (device_type, device_id) DO UPDATE SET
			count = app.device_fail_counts.count + 1,
			last_fail_at = NOW(),
			updated_at = NOW()
		RETURNING count`
	err := DB.Raw(sql, deviceType, deviceID).Scan(&fc).Error
	return fc.Count, err
}

func (s *PostgresDeviceFailStore) Reset(deviceType, deviceID string) error {
	return DB.Exec(
		"UPDATE app.device_fail_counts SET count = 0, updated_at = NOW() WHERE device_type = ? AND device_id = ?",
		deviceType, deviceID,
	).Error
}

func (s *PostgresDeviceFailStore) Get(deviceType, deviceID string) (int, error) {
	var fc model.DeviceFailCount
	err := DB.Where("device_type = ? AND device_id = ?", deviceType, deviceID).First(&fc).Error
	if err == gorm.ErrRecordNotFound {
		return 0, nil
	}
	return fc.Count, err
}
