package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"promthus/internal/config"
	"promthus/internal/handler"
	"promthus/internal/kms"
	"promthus/internal/logger"
	"promthus/internal/metrics"
	"promthus/internal/middleware"
	"promthus/internal/mq"
	"promthus/internal/repository"
	"promthus/internal/router"
	"promthus/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()
	// 初始化日志,并且在return前调用Sync()刷写日志到文件或者输出;
	logger.Init(cfg.Server.Mode)
	defer logger.Sync()

	gin.SetMode(cfg.Server.Mode)
	// repository包名,进行DB初始化;
	repository.InitDB(&cfg.Database)
	defer repository.CloseDB()

	kms.Init(cfg.KMS.MasterKeyPath)

	middleware.SetTokenSecret(cfg.Auth.TokenSecret)

	metrics.Init()

	var publisher *mq.Publisher
	var auditConsumer *mq.AuditConsumer
	// 初始化戏哦啊西队列生产者;
	pub, err := mq.NewPublisher(cfg.RabbitMQ.URL)
	if err != nil {
		logger.Warn("RabbitMQ not available, running without message queue", zap.Error(err))
	} else {
		publisher = pub
		defer publisher.Close()

		// 初始化队列消费者;
		consumer, err := mq.NewAuditConsumer(cfg.RabbitMQ.URL, 3)
		if err != nil {
			logger.Warn("failed to start audit consumer", zap.Error(err))
		} else {
			auditConsumer = consumer
			defer auditConsumer.Close()
		}
	}

	// 初始化会话存储,这里做通配处理,后续可以拓展成readis之类的其他存储类型;
	sessionStore := repository.NewPostgresSessionStore()
	// 初始化设备失败计数存储,这里做通配处理,后续可以拓展成readis之类的其他存储类型;
	failStore := repository.NewPostgresDeviceFailStore()

	authSvc := service.NewAuthService(sessionStore, &cfg.Auth)
	lockSvc := service.NewLockService(failStore, publisher)
	adminSvc := service.NewAdminService(sessionStore)

	authHandler := handler.NewAuthHandler(authSvc)
	lockHandler := handler.NewLockHandler(lockSvc)
	adminHandler := handler.NewAdminHandler(adminSvc)

	// 初始化路由,注册handler,用于gin路由控制;
	r := router.Setup(authHandler, lockHandler, adminHandler)

	r.Use(metrics.PrometheusMiddleware())
	r.GET("/metrics", metrics.MetricsHandler())

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// 启动http服务器,监听端口,并启动一个goroutine来处理请求;
	go func() {
		logger.Info("server starting", zap.String("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()

	// 启动一个goroutine来处理会话清理;
	go startSessionCleaner(sessionStore)
	// 监听信号,SIGINT,SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	// 等待信号,如果收到信号,则退出;
	<-quit

	logger.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", zap.Error(err))
	}

	logger.Info("server exited")
}

func startSessionCleaner(store repository.SessionStore) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		count, err := store.CleanExpired()
		if err != nil {
			logger.Error("session cleanup failed", zap.Error(err))
		} else if count > 0 {
			logger.Info("cleaned expired sessions", zap.Int64("count", count))
		}
	}
}
