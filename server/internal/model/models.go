package model

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DeviceTypeLock 当前业务仅锁具，后续扩展传感器等时在 device_types 注册
const DeviceTypeLock = "lock"

// ==================== 用户表 app.users ====================

type User struct {
	ID           int64          `gorm:"primaryKey;autoIncrement" json:"-"`
	UUID         uuid.UUID      `gorm:"type:uuid;not null;default:gen_random_uuid();uniqueIndex:idx_users_uuid" json:"uuid"`
	Phone        string         `gorm:"type:varchar(20);not null" json:"-"`
	PhoneMasked  string         `gorm:"-" json:"phone,omitempty"`
	PasswordHash string         `gorm:"type:varchar(100);not null" json:"-"`
	Name         string         `gorm:"type:varchar(50);not null" json:"name"`
	Department   sql.NullString `gorm:"type:varchar(100)" json:"department"`
	Role         string         `gorm:"type:varchar(20);not null" json:"role"`
	Status       int16          `gorm:"type:smallint;not null;default:1" json:"status"`
	CreatedAt    time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string { return "app.users" }

// ==================== 会话表 app.sessions ====================

type Session struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"-"`
	JTI       uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_sessions_jti" json:"jti"`
	UserID    int64     `gorm:"not null;index:idx_sessions_user_id" json:"user_id"`
	Role      string    `gorm:"type:varchar(20);not null" json:"role"`
	ExpiresAt time.Time `gorm:"not null;index:idx_sessions_expires" json:"expires_at"`
	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UserAgent string    `gorm:"type:varchar(200)" json:"user_agent"`
	IPAddress string    `gorm:"type:varchar(45)" json:"ip_address"`
}

func (Session) TableName() string { return "app.sessions" }

// ==================== 锁具设备表 app.devices_lock ====================

type Device struct {
	ID            int64          `gorm:"primaryKey;autoIncrement" json:"-"`
	DeviceID      string         `gorm:"type:varchar(32);not null" json:"device_id"`
	Name          string         `gorm:"type:varchar(100);not null" json:"name"`
	LocationText  string         `gorm:"type:text;not null" json:"location_text"`
	Longitude     *float64       `gorm:"type:numeric(10,7)" json:"longitude,omitempty"`
	Latitude      *float64       `gorm:"type:numeric(10,7)" json:"latitude,omitempty"`
	PipelineTag   sql.NullString `gorm:"type:varchar(50)" json:"pipeline_tag"`
	RiskLevel     int16          `gorm:"type:smallint;not null;default:1" json:"risk_level"`
	KeyEncrypted  []byte         `gorm:"type:bytea;not null" json:"-"`
	KeyVersion    int16          `gorm:"type:smallint;not null;default:1" json:"key_version"`
	Status        int16          `gorm:"type:smallint;not null;default:1" json:"status"`
	LastActiveAt  *time.Time     `gorm:"" json:"last_active_at,omitempty"`
	CreatedAt     time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Device) TableName() string { return "app.devices_lock" }

// ==================== 权限授权表 app.permissions (device_type + device_id) ====================

type Permission struct {
	ID         int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     int64      `gorm:"not null;index:idx_permissions_user_id" json:"user_id"`
	DeviceType string     `gorm:"type:varchar(32);not null;index:idx_permissions_device" json:"device_type"`
	DeviceID   string     `gorm:"type:varchar(32);not null;index:idx_permissions_device" json:"device_id"`
	GrantedBy  int64      `gorm:"not null" json:"granted_by"`
	ValidFrom  time.Time  `gorm:"not null" json:"valid_from"`
	ValidUntil *time.Time `gorm:"" json:"valid_until,omitempty"`
	Status     int16      `gorm:"type:smallint;not null;default:1" json:"status"`
	RevokedBy  *int64     `gorm:"" json:"revoked_by,omitempty"`
	RevokedAt  *time.Time `gorm:"" json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `gorm:"not null;default:now()" json:"created_at"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (Permission) TableName() string { return "app.permissions" }

// ==================== 审计日志表 log.audit_logs ====================

type AuditLog struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      int64     `gorm:"not null" json:"user_id"`
	DeviceID    string    `gorm:"type:varchar(32);not null" json:"device_id"`
	DeviceType  string    `gorm:"type:varchar(32);not null;default:lock" json:"device_type"`
	Action      string    `gorm:"type:varchar(30);not null" json:"action"`
	ResultCode  int16     `gorm:"type:smallint;not null" json:"result_code"`
	ClientIP    string    `gorm:"type:varchar(45);not null" json:"client_ip"`
	DeviceModel string    `gorm:"type:varchar(100)" json:"device_model"`
	Extra       JSON      `gorm:"type:jsonb" json:"extra,omitempty"`
	OccurredAt  time.Time `gorm:"not null" json:"occurred_at"`
}

func (AuditLog) TableName() string { return "log.audit_logs" }

// ==================== 限流表 app.rate_limits ====================

type RateLimit struct {
	Key         string    `gorm:"type:varchar(150);primaryKey" json:"key"`
	Count       int       `gorm:"not null;default:1" json:"count"`
	WindowStart time.Time `gorm:"not null" json:"window_start"`
	UpdatedAt   time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

func (RateLimit) TableName() string { return "app.rate_limits" }

// ==================== 告警表 app.alerts ====================

type Alert struct {
	ID         int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	AlertType  string     `gorm:"type:varchar(40);not null" json:"alert_type"`
	DeviceType string     `gorm:"type:varchar(32);not null;default:lock" json:"device_type"`
	DeviceID   string     `gorm:"type:varchar(32);not null" json:"device_id"`
	UserID     *int64     `gorm:"" json:"user_id,omitempty"`
	Severity   int16      `gorm:"type:smallint;not null" json:"severity"`
	Status     int16      `gorm:"type:smallint;not null;default:0" json:"status"`
	HandledBy  *int64     `gorm:"" json:"handled_by,omitempty"`
	HandleNote *string    `gorm:"type:text" json:"handle_note,omitempty"`
	Extra      JSON       `gorm:"type:jsonb" json:"extra,omitempty"`
	CreatedAt  time.Time  `gorm:"not null;default:now()" json:"created_at"`
	HandledAt  *time.Time `gorm:"" json:"handled_at,omitempty"`
}

func (Alert) TableName() string { return "app.alerts" }

// ==================== 操作日志表 log.operation_logs ====================

type OperationLog struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	OperatorID     int64     `gorm:"not null" json:"operator_id"`
	Action         string    `gorm:"type:varchar(50);not null" json:"action"`
	TargetType     string    `gorm:"type:varchar(20);not null" json:"target_type"`
	TargetID       int64     `gorm:"not null" json:"target_id"`
	BeforeSnapshot JSON      `gorm:"type:jsonb" json:"before_snapshot,omitempty"`
	AfterSnapshot  JSON      `gorm:"type:jsonb" json:"after_snapshot,omitempty"`
	OccurredAt     time.Time `gorm:"not null;default:now()" json:"occurred_at"`
}

func (OperationLog) TableName() string { return "log.operation_logs" }

// ==================== 连续失败计数表 app.device_fail_counts (device_type, device_id) ====================

type DeviceFailCount struct {
	DeviceType  string    `gorm:"type:varchar(32);primaryKey" json:"device_type"`
	DeviceID    string    `gorm:"type:varchar(32);primaryKey" json:"device_id"`
	Count       int       `gorm:"not null;default:0" json:"count"`
	LastFailAt  time.Time `gorm:"not null" json:"last_fail_at"`
	UpdatedAt   time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

func (DeviceFailCount) TableName() string { return "app.device_fail_counts" }

// ==================== IP 封锁表 app.ip_blocks ====================

type IPBlock struct {
	IP        string    `gorm:"type:varchar(45);primaryKey" json:"ip"`
	BlockedAt time.Time `gorm:"not null;default:now()" json:"blocked_at"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	Reason    string    `gorm:"type:varchar(100)" json:"reason"`
}

func (IPBlock) TableName() string { return "app.ip_blocks" }
