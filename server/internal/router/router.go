package router

import (
	"promthus/internal/handler"
	"promthus/internal/middleware"

	"github.com/gin-gonic/gin"
)

func Setup(
	authHandler *handler.AuthHandler,
	lockHandler *handler.LockHandler,
	adminHandler *handler.AdminHandler,
) *gin.Engine {
	// 根据注册函数进行gin引擎注册,随后返回注册好的gin引擎;
	r := gin.New()

	// 使用中间件链,Recovery,RequestID,SecurityHeaders,CORS,AccessLog,GlobalRateLimit;
	r.Use(
		middleware.Recovery(),
		middleware.RequestID(),
		middleware.SecurityHeaders(),
		middleware.CORS(),
		middleware.AccessLog(),
		middleware.GlobalRateLimit(),
	)

	// GET会检测当前库是否正常;
	r.GET("/api/health", handler.Health)

	// 认证组,POST登录,POST登出;
	auth := r.Group("/api/auth")
	{
		// 登录可以多handle,次序执行;
		auth.POST("/login", middleware.LoginRateLimit(), authHandler.Login)
		auth.POST("/logout", middleware.Auth(), authHandler.Logout)
	}

	// 锁具组,所有接口都需要认证认证;
	lock := r.Group("/api/lock").Use(middleware.Auth())
	{
		lock.GET("/devices", lockHandler.GetDevices)
		lock.POST("/challenge", lockHandler.Challenge)
		lock.POST("/report", lockHandler.Report)
	}

	admin := r.Group("/api/admin").Use(middleware.Auth(), middleware.RBAC("admin"))
	{
		admin.GET("/dashboard", adminHandler.Dashboard)

		admin.GET("/users", adminHandler.ListUsers)
		admin.POST("/users", adminHandler.CreateUser)
		admin.PUT("/users/:uuid", adminHandler.UpdateUser)
		admin.POST("/users/:uuid/reset-pwd", adminHandler.ResetPassword)

		admin.GET("/devices", adminHandler.ListDevices)
		admin.POST("/devices", adminHandler.CreateDevice)

		admin.GET("/permissions", adminHandler.ListPermissions)
		admin.POST("/permissions", adminHandler.GrantPermission)
		admin.POST("/permissions/batch", adminHandler.BatchGrantPermissions)
		admin.DELETE("/permissions/:id", adminHandler.RevokePermission)

		admin.GET("/audit-logs", adminHandler.ListAuditLogs)

		admin.GET("/alerts", adminHandler.ListAlerts)
		admin.PUT("/alerts/:id", adminHandler.HandleAlert)
	}

	return r
}
