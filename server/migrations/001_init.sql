-- Migration 001: 数据库基线（唯一）
-- 依据 docs/database/01-数据库总体设计.md，不兼容任何旧格式。
-- Promthus NFC 智能管控系统

BEGIN;

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ==================== Schemas ====================
CREATE SCHEMA app;
CREATE SCHEMA log;
CREATE SCHEMA metrics;

-- ==================== app.device_types ====================
CREATE TABLE app.device_types (
    id          SMALLSERIAL PRIMARY KEY,
    code        VARCHAR(32) NOT NULL UNIQUE,
    name        VARCHAR(100) NOT NULL,
    table_name  VARCHAR(64) NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO app.device_types (code, name, table_name) VALUES
('lock', 'NFC锁具', 'devices_lock');

-- ==================== app.users ====================
CREATE TABLE app.users (
    id            BIGSERIAL PRIMARY KEY,
    uuid          UUID NOT NULL DEFAULT gen_random_uuid(),
    phone         VARCHAR(20) NOT NULL,
    password_hash VARCHAR(100) NOT NULL,
    name          VARCHAR(50) NOT NULL,
    department    VARCHAR(100),
    role          VARCHAR(20) NOT NULL,
    status        SMALLINT NOT NULL DEFAULT 1,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_users_uuid        ON app.users(uuid);
CREATE UNIQUE INDEX idx_users_phone       ON app.users(phone) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_status_role       ON app.users(status, role) WHERE deleted_at IS NULL;

-- ==================== app.sessions ====================
CREATE TABLE app.sessions (
    id         BIGSERIAL PRIMARY KEY,
    jti        UUID NOT NULL,
    user_id    BIGINT NOT NULL REFERENCES app.users(id),
    role       VARCHAR(20) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_agent VARCHAR(200),
    ip_address INET
);

CREATE UNIQUE INDEX idx_sessions_jti      ON app.sessions(jti);
CREATE INDEX idx_sessions_user_id         ON app.sessions(user_id);
CREATE INDEX idx_sessions_expires         ON app.sessions(expires_at);

-- ==================== app.devices_lock ====================
CREATE TABLE app.devices_lock (
    id              BIGSERIAL PRIMARY KEY,
    device_id       VARCHAR(32) NOT NULL,
    name            VARCHAR(100) NOT NULL,
    location_text   TEXT NOT NULL,
    longitude       NUMERIC(10,7),
    latitude        NUMERIC(10,7),
    pipeline_tag    VARCHAR(50),
    risk_level      SMALLINT NOT NULL DEFAULT 1,
    key_encrypted   BYTEA NOT NULL,
    key_version     SMALLINT NOT NULL DEFAULT 1,
    status          SMALLINT NOT NULL DEFAULT 1,
    last_active_at  TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_devices_lock_device_id ON app.devices_lock(device_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_devices_lock_pipeline ON app.devices_lock(pipeline_tag);
CREATE INDEX idx_devices_lock_status   ON app.devices_lock(status) WHERE deleted_at IS NULL;

-- ==================== app.permissions ====================
CREATE TABLE app.permissions (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES app.users(id),
    device_type VARCHAR(32) NOT NULL,
    device_id   VARCHAR(32) NOT NULL,
    granted_by  BIGINT NOT NULL REFERENCES app.users(id),
    valid_from  TIMESTAMPTZ NOT NULL,
    valid_until TIMESTAMPTZ,
    status      SMALLINT NOT NULL DEFAULT 1,
    revoked_by  BIGINT REFERENCES app.users(id),
    revoked_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_permissions_user_device
    ON app.permissions(user_id, device_type, device_id) WHERE status = 1;
CREATE INDEX idx_permissions_user_id     ON app.permissions(user_id);
CREATE INDEX idx_permissions_device      ON app.permissions(device_type, device_id);
CREATE INDEX idx_permissions_valid_until ON app.permissions(valid_until) WHERE valid_until IS NOT NULL AND status = 1;

-- ==================== app.alerts ====================
CREATE TABLE app.alerts (
    id          BIGSERIAL PRIMARY KEY,
    alert_type  VARCHAR(40) NOT NULL,
    device_type VARCHAR(32) NOT NULL DEFAULT 'lock',
    device_id   VARCHAR(32) NOT NULL,
    user_id     BIGINT REFERENCES app.users(id),
    severity    SMALLINT NOT NULL,
    status      SMALLINT NOT NULL DEFAULT 0,
    handled_by  BIGINT REFERENCES app.users(id),
    handle_note TEXT,
    extra       JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    handled_at  TIMESTAMPTZ
);

CREATE INDEX idx_alerts_status          ON app.alerts(status) WHERE status = 0;
CREATE INDEX idx_alerts_device_created   ON app.alerts(device_id, created_at DESC);
CREATE INDEX idx_alerts_device_type     ON app.alerts(device_type);
CREATE INDEX idx_alerts_severity_status  ON app.alerts(severity, status);

-- ==================== app.rate_limits ====================
CREATE TABLE app.rate_limits (
    key          VARCHAR(150) PRIMARY KEY,
    count        INT NOT NULL DEFAULT 1,
    window_start TIMESTAMPTZ NOT NULL,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ==================== app.device_fail_counts ====================
CREATE TABLE app.device_fail_counts (
    device_type   VARCHAR(32) NOT NULL,
    device_id     VARCHAR(32) NOT NULL,
    count         INT NOT NULL DEFAULT 0,
    last_fail_at  TIMESTAMPTZ NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (device_type, device_id)
);

CREATE INDEX idx_device_fail_counts_type ON app.device_fail_counts(device_type);

-- ==================== app.ip_blocks ====================
CREATE TABLE app.ip_blocks (
    ip          INET PRIMARY KEY,
    blocked_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL,
    reason      VARCHAR(100)
);

CREATE INDEX idx_ip_blocks_expires ON app.ip_blocks(expires_at);

-- ==================== log.audit_logs ====================
CREATE TABLE log.audit_logs (
    id           BIGSERIAL,
    user_id      BIGINT NOT NULL,
    device_id    VARCHAR(32) NOT NULL,
    device_type  VARCHAR(32) NOT NULL DEFAULT 'lock',
    action       VARCHAR(30) NOT NULL,
    result_code  SMALLINT NOT NULL,
    client_ip    INET NOT NULL,
    device_model VARCHAR(100),
    extra        JSONB,
    occurred_at  TIMESTAMPTZ NOT NULL
) PARTITION BY RANGE (occurred_at);

CREATE TABLE log.audit_logs_2026_02 PARTITION OF log.audit_logs
    FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
CREATE TABLE log.audit_logs_2026_03 PARTITION OF log.audit_logs
    FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');
CREATE TABLE log.audit_logs_2026_04 PARTITION OF log.audit_logs
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');

CREATE INDEX idx_audit_logs_user_id     ON log.audit_logs(user_id);
CREATE INDEX idx_audit_logs_device_id   ON log.audit_logs(device_id);
CREATE INDEX idx_audit_logs_occurred_at ON log.audit_logs(occurred_at);
CREATE INDEX idx_audit_logs_action      ON log.audit_logs(action);
CREATE INDEX idx_audit_logs_device_type ON log.audit_logs(device_type);

-- ==================== log.operation_logs ====================
CREATE TABLE log.operation_logs (
    id              BIGSERIAL PRIMARY KEY,
    operator_id     BIGINT NOT NULL,
    action          VARCHAR(50) NOT NULL,
    target_type     VARCHAR(20) NOT NULL,
    target_id       BIGINT NOT NULL,
    before_snapshot JSONB,
    after_snapshot  JSONB,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_operation_logs_operator ON log.operation_logs(operator_id);
CREATE INDEX idx_operation_logs_action   ON log.operation_logs(action);
CREATE INDEX idx_operation_logs_occurred ON log.operation_logs(occurred_at);

-- ==================== 初始管理员 ====================
-- 手机号: 13800000000  密码: Admin@2026
INSERT INTO app.users (uuid, phone, password_hash, name, role, status)
VALUES (
    gen_random_uuid(),
    '13800000000',
    '$argon2id$v=19$m=65536,t=3,p=4$z5GDtRfU5NJv00nDL8bsrA$8sSciyxn7MWCVEIISgbczNIgPOoSq9oqipaLxkBgbQ0',
    '系统管理员',
    'admin',
    1
);

COMMIT;
