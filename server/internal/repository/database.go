package repository

import (
	"fmt"
	"time"

	"promthus/internal/config"
	"promthus/internal/logger"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// 全局DB实例;
var DB *gorm.DB

func InitDB(cfg *config.DatabaseConfig) {
	var err error
	// 打开一个db;
	DB, err = gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger:                 gormlogger.Default.LogMode(gormlogger.Warn),
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
	})
	if err != nil {
		logger.Fatal("failed to connect database", zap.Error(err))
	}

	sqlDB, err := DB.DB()
	if err != nil {
		logger.Fatal("failed to get sql.DB", zap.Error(err))
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	if err = sqlDB.Ping(); err != nil {
		logger.Fatal("database ping failed, refusing to start", zap.Error(err))
	}

	logger.Info("database connected",
		zap.String("host", cfg.Host),
		zap.String("db", cfg.DBName),
	)
}

func CloseDB() {
	if DB != nil {
		sqlDB, _ := DB.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	}
}

// 接收一个函数,类似于函数指针,返回error,相当于begin和commit由grom来实现,自己关心业务逻辑即可;
func Transaction(fn func(tx *gorm.DB) error) error {
	return DB.Transaction(fn)
}

// 进行事务重试,如果失败,则重试;
func RetryTransaction(maxRetries int, fn func(tx *gorm.DB) error) error {
	var err error
	for i := 0; i <= maxRetries; i++ {
		err = DB.Transaction(fn)
		if err == nil {
			return nil
		}
		if i < maxRetries {
			time.Sleep(time.Duration(50*(i+1)) * time.Millisecond)
		}
	}
	return fmt.Errorf("transaction failed after %d retries: %w", maxRetries, err)
}
