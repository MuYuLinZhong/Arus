package handler

import (
	"net/http"

	"promthus/internal/model"
	"promthus/internal/repository"

	"github.com/gin-gonic/gin"
)

func Health(c *gin.Context) {
	sqlDB, err := repository.DB.DB()
	if err != nil {
		model.Fail(c, http.StatusServiceUnavailable, model.CodeInternalError, "database unavailable")
		return
	}

	if err := sqlDB.Ping(); err != nil {
		model.Fail(c, http.StatusServiceUnavailable, model.CodeInternalError, "database ping failed")
		return
	}

	model.OK(c, gin.H{
		"status":  "healthy",
		"service": "lock-service",
	})
}
