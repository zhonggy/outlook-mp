// Package middleware Gin 中间件：JWT 认证、API Key 认证、CORS。
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"outlook-manager/internal/pkg"
	"outlook-manager/internal/repository"
)

// JWTAuth 校验 Authorization: Bearer <jwt>。
func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		token := strings.TrimPrefix(header, "Bearer ")
		if token == "" || token == header {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "缺少认证令牌"})
			return
		}
		claims, err := pkg.ParseToken(secret, token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "令牌无效或已过期"})
			return
		}
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

// APIKeyAuth 校验 X-API-Key 头（自动化对接用）。
func APIKeyAuth(repo *repository.APIKeyRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-API-Key")
		if key == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "缺少 X-API-Key"})
			return
		}
		k, err := repo.ByKey(key)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "API Key 无效"})
			return
		}
		repo.TouchUsed(k.ID)
		c.Set("apiKeyName", k.Name)
		c.Next()
	}
}

// CORS 允许前后端分离开发（生产同域部署时无害）。
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			origin = "*"
		}
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization,X-API-Key")
		c.Header("Access-Control-Max-Age", "86400")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
