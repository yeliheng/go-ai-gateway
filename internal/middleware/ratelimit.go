package middleware

import (
	"ai-gateway/common/logger"
	"ai-gateway/internal/limiter"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

var limit *limiter.Limiter

func InitRateLimit() {
	limit = limiter.NewLimiter()
}

func RateLimitMiddleware() gin.HandlerFunc {

	if limit == nil {
		InitRateLimit()
	}

	return func(c *gin.Context) {

		ip := c.ClientIP()
		userID := ""

		// 没用的话后面删除
		if val, exists := c.Get("userID"); exists {
			userID = val.(string)
		} else {
		}

		allowed, err := limit.Check(c.Request.Context(), c.Request.URL.Path, c.Request.Method, ip, userID)
		if err != nil {
			logger.Log.Error("RateLimit check failed", zap.Error(err))
			c.Next()
			return
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Too Many Requests"})
			return
		}

		c.Next()
	}
}

func SetUserID(c *gin.Context, claims jwt.MapClaims) {
	if sub, ok := claims["sub"]; ok {
		c.Set("userID", fmt.Sprint(sub))
	}
}
