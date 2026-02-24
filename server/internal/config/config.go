package config

import (
	"os"
	"strconv"
	"time"
)

// 顶层配置文件
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	RabbitMQ RabbitMQConfig
	Auth     AuthConfig
	KMS      KMSConfig
}

// http服务配置
type ServerConfig struct {
	Port string
	Mode string // debug / release
	// 读写最长等待时间
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// 数据库连接驱动配置
type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// 消息队列配置
type RabbitMQConfig struct {
	URL string
}

type AuthConfig struct {
	TokenSecret   string        // HMAC 签 Token 的密钥
	SessionTTL    time.Duration // 登录后 Session 有效期，如 8 小时
	Argon2Memory  uint32        // Argon2id 内存参数（KB）
	Argon2Time    uint32        // 迭代次数
	Argon2Threads uint8         // 并行度
}

type KMSConfig struct {
	MasterKeyPath string // 主密钥文件路径，如 "./master.key"
	Provider      string // "local" | "aliyun" | "vault"，当前只用 local
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         envOrDefault("SERVER_PORT", "8080"),
			Mode:         envOrDefault("GIN_MODE", "debug"),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		Database: DatabaseConfig{
			Host:            envOrDefault("DB_HOST", "localhost"),
			Port:            envOrDefault("DB_PORT", "5432"),
			User:            envOrDefault("DB_USER", "promthus"),
			Password:        envOrDefault("DB_PASSWORD", "promthus"),
			DBName:          envOrDefault("DB_NAME", "promthus"),
			SSLMode:         envOrDefault("DB_SSLMODE", "disable"),
			MaxOpenConns:    envOrDefaultInt("DB_MAX_OPEN_CONNS", 50),
			MaxIdleConns:    envOrDefaultInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: time.Hour,
			ConnMaxIdleTime: 30 * time.Minute,
		},
		RabbitMQ: RabbitMQConfig{
			URL: envOrDefault("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		},
		Auth: AuthConfig{
			TokenSecret:   envOrDefault("AUTH_TOKEN_SECRET", "change-me-in-production"),
			SessionTTL:    8 * time.Hour,
			Argon2Memory:  65536,
			Argon2Time:    3,
			Argon2Threads: 4,
		},
		KMS: KMSConfig{
			MasterKeyPath: envOrDefault("KMS_MASTER_KEY_PATH", "./master.key"),
			Provider:      envOrDefault("KMS_PROVIDER", "local"),
		},
	}
}

// 很他妈奇怪的方法,意思是给一个类型整了一个可以调用的方法,相当于变量.方法();
func (c *DatabaseConfig) DSN() string {
	return "host=" + c.Host +
		" port=" + c.Port +
		" user=" + c.User +
		" password=" + c.Password +
		" dbname=" + c.DBName +
		" sslmode=" + c.SSLMode +
		" TimeZone=Asia/Shanghai"
}

// 从环境变量获取key值,如果没获取到,返回fallback
func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// 从环境变量获取key值,转化成int,如果没获取到,返回fallback
func envOrDefaultInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
