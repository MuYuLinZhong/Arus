package service

import (
	"errors"
	"time"

	"promthus/internal/config"
	"promthus/internal/crypto"
	"promthus/internal/logger"
	"promthus/internal/middleware"
	"promthus/internal/model"
	"promthus/internal/repository"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AuthService struct {
	sessionStore repository.SessionStore
	cfg          *config.AuthConfig
}

func NewAuthService(ss repository.SessionStore, cfg *config.AuthConfig) *AuthService {
	return &AuthService{sessionStore: ss, cfg: cfg}
}

type LoginRequest struct {
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	UserUUID  string    `json:"user_uuid"`
	Role      string    `json:"role"`
	Name      string    `json:"name"`
}

func (s *AuthService) Login(req *LoginRequest, userAgent, ipAddress string) (*LoginResponse, int, string) {
	logger.Info("login: attempt",
		zap.String("phone", req.Phone), zap.String("ip", ipAddress), zap.String("user_agent", userAgent))

	var user model.User
	err := repository.DB.Where("phone = ? AND deleted_at IS NULL", req.Phone).First(&user).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		crypto.DummyVerify()
		logger.Info("login: failed, user not found (dummy verify executed)", zap.String("phone", req.Phone))
		return nil, model.CodeAuthFailed, "invalid credentials"
	}
	if err != nil {
		logger.Error("login: db query failed", zap.Error(err))
		return nil, model.CodeInternalError, "internal error"
	}

	valid, err := crypto.VerifyPassword(req.Password, user.PasswordHash)
	if err != nil || !valid {
		logger.Info("login: failed, invalid password",
			zap.String("phone", req.Phone), zap.Int64("user_id", user.ID))
		return nil, model.CodeAuthFailed, "invalid credentials"
	}

	if user.Status == 0 {
		logger.Info("login: rejected, account disabled",
			zap.String("phone", req.Phone), zap.Int64("user_id", user.ID))
		return nil, model.CodeAccountDisabled, "account has been disabled"
	}

	jti := uuid.New()
	expiresAt := time.Now().Add(s.cfg.SessionTTL)

	session := &model.Session{
		JTI:       jti,
		UserID:    user.ID,
		Role:      user.Role,
		ExpiresAt: expiresAt,
		UserAgent: userAgent,
		IPAddress: ipAddress,
	}

	if err := s.sessionStore.Create(session); err != nil {
		logger.Error("login: create session failed", zap.Error(err), zap.Int64("user_id", user.ID))
		return nil, model.CodeInternalError, "internal error"
	}

	token, err := middleware.GenerateToken(user.UUID, jti)
	if err != nil {
		logger.Error("login: generate token failed", zap.Error(err), zap.Int64("user_id", user.ID))
		return nil, model.CodeInternalError, "internal error"
	}

	logger.Info("login: success",
		zap.Int64("user_id", user.ID), zap.String("role", user.Role),
		zap.String("ip", ipAddress), zap.Time("expires_at", expiresAt))

	return &LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		UserUUID:  user.UUID.String(),
		Role:      user.Role,
		Name:      user.Name,
	}, 0, ""
}

func (s *AuthService) Logout(jti uuid.UUID) error {
	logger.Info("logout: session invalidated", zap.String("jti", jti.String()))
	return s.sessionStore.DeleteByJTI(jti)
}
