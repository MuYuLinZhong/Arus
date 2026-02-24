package middleware

import (
	"net/http"
	"strings"
	"time"

	"promthus/internal/model"
	"promthus/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

/*
检查 Header 里有没有合法 Token → 用 Token 里的 JTI 查 session 是否有效 → 再查用户是否存在且未禁用；
任何一步不通过就 401 并 Abort；
全部通过就把当前用户信息写入 Context 并 Next，让后续逻辑按「已登录用户」继续跑。
*/
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			model.Fail(c, http.StatusUnauthorized, model.CodeSessionExpired, "missing or invalid authorization header")
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := ParseToken(tokenStr)
		if err != nil {
			model.Fail(c, http.StatusUnauthorized, model.CodeSessionExpired, "invalid token")
			c.Abort()
			return
		}

		var session model.Session
		result := repository.DB.
			Where("jti = ? AND expires_at > ?", claims.JTI, time.Now()).
			First(&session)
		if result.Error != nil {
			model.Fail(c, http.StatusUnauthorized, model.CodeSessionExpired, "session expired, please login again")
			c.Abort()
			return
		}

		var user model.User
		result = repository.DB.Where("id = ? AND deleted_at IS NULL", session.UserID).First(&user)
		if result.Error != nil || user.Status == 0 {
			repository.DB.Where("user_id = ?", session.UserID).Delete(&model.Session{})
			model.Fail(c, http.StatusUnauthorized, model.CodeAccountDisabled, "account has been disabled")
			c.Abort()
			return
		}

		c.Set("user_id", session.UserID)
		c.Set("user_uuid", claims.UserUUID)
		c.Set("role", session.Role)
		c.Next()
	}
}

func RBAC(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			model.Fail(c, http.StatusForbidden, model.CodeForbidden, "access denied")
			c.Abort()
			return
		}

		roleStr := role.(string)
		for _, allowed := range allowedRoles {
			if roleStr == allowed {
				c.Next()
				return
			}
		}

		model.Fail(c, http.StatusForbidden, model.CodeForbidden, "insufficient permissions")
		c.Abort()
	}
}

// Token claims and helpers

type TokenClaims struct {
	UserUUID uuid.UUID
	JTI      uuid.UUID
}

var tokenSecret []byte

func SetTokenSecret(secret string) {
	tokenSecret = []byte(secret)
}

func GenerateToken(userUUID, jti uuid.UUID) (string, error) {
	payload := userUUID.String() + ":" + jti.String()
	return signHMAC(payload, tokenSecret)
}

func ParseToken(tokenStr string) (*TokenClaims, error) {
	payload, err := verifyHMAC(tokenStr, tokenSecret)
	if err != nil {
		return nil, err
	}

	parts := strings.SplitN(payload, ":", 2)
	if len(parts) != 2 {
		return nil, ErrInvalidToken
	}

	userUUID, err := uuid.Parse(parts[0])
	if err != nil {
		return nil, ErrInvalidToken
	}

	jti, err := uuid.Parse(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}

	return &TokenClaims{
		UserUUID: userUUID,
		JTI:      jti,
	}, nil
}
