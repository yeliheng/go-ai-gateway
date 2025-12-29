package middleware

import (
	"fmt"
	"net/http"

	"github.com/yeliheng/go-ai-gateway/common/config"
	"github.com/yeliheng/go-ai-gateway/internal/cache"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func WebSocketAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.Query("token")
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(config.GlobalConfig.JWT.Secret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		valid, err := cache.ValidateToken(c.Request.Context(), tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal auth error"})
			return
		}
		if !valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token expired or revoked"})
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			SetUserID(c, claims)
		}

		c.Next()
	}
}
