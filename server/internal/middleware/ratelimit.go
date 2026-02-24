package middleware

import (
	"fmt"
	"net/http"
	"time"

	"promthus/internal/model"
	"promthus/internal/repository"

	"github.com/gin-gonic/gin"
)

type RateLimitConfig struct {
	KeyFunc    func(c *gin.Context) string
	Limit      int
	WindowSecs int
}

func RateLimiter(cfg RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := checkIPBlock(c); err != nil {
			model.Fail(c, http.StatusForbidden, model.CodeTooManyRequests, "IP has been temporarily blocked")
			c.Abort()
			return
		}

		key := cfg.KeyFunc(c)
		count, err := incrementRateLimit(key, cfg.WindowSecs)
		if err != nil {
			c.Next()
			return
		}

		if count > cfg.Limit {
			c.Header("Retry-After", fmt.Sprintf("%d", cfg.WindowSecs))
			model.Fail(c, http.StatusTooManyRequests, model.CodeTooManyRequests, "too many requests")
			c.Abort()
			return
		}

		c.Next()
	}
}

func checkIPBlock(c *gin.Context) error {
	var count int64
	repository.DB.Model(&model.IPBlock{}).
		Where("ip = ? AND expires_at > ?", c.ClientIP(), time.Now()).
		Count(&count)
	if count > 0 {
		return fmt.Errorf("ip blocked")
	}
	return nil
}

func incrementRateLimit(key string, windowSecs int) (int, error) {
	var rl model.RateLimit
	sql := `INSERT INTO app.rate_limits (key, count, window_start, updated_at)
		VALUES (?, 1, NOW(), NOW())
		ON CONFLICT (key) DO UPDATE SET
			count = CASE
				WHEN app.rate_limits.window_start < NOW() - INTERVAL '%d seconds'
				THEN 1
				ELSE app.rate_limits.count + 1
			END,
			window_start = CASE
				WHEN app.rate_limits.window_start < NOW() - INTERVAL '%d seconds'
				THEN NOW()
				ELSE app.rate_limits.window_start
			END,
			updated_at = NOW()
		RETURNING count, window_start`

	formattedSQL := fmt.Sprintf(sql, windowSecs, windowSecs)
	result := repository.DB.Raw(formattedSQL, key).Scan(&rl)
	if result.Error != nil {
		return 0, result.Error
	}
	return rl.Count, nil
}

// LoginRateLimit is a specialized rate limiter for login endpoint with IP blocking.
func LoginRateLimit() gin.HandlerFunc {
	return RateLimiter(RateLimitConfig{
		KeyFunc: func(c *gin.Context) string {
			return "login:ip:" + c.ClientIP()
		},
		Limit:      10,
		WindowSecs: 60,
	})
}

// ChallengeRateLimit limits challenge requests per device. Key includes device_type per DB design.
func ChallengeRateLimit() gin.HandlerFunc {
	return RateLimiter(RateLimitConfig{
		KeyFunc: func(c *gin.Context) string {
			deviceID := c.PostForm("device_id")
			if deviceID == "" {
				deviceID = "unknown"
			}
			return "challenge:lock:" + deviceID
		},
		Limit:      5,
		WindowSecs: 60,
	})
}

// GlobalRateLimit applies a general rate limit per IP.
func GlobalRateLimit() gin.HandlerFunc {
	return RateLimiter(RateLimitConfig{
		KeyFunc: func(c *gin.Context) string {
			return "global:ip:" + c.ClientIP()
		},
		Limit:      100,
		WindowSecs: 60,
	})
}
