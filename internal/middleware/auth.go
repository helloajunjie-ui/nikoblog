package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"nikoblog/internal/utils"
)

// AuthMiddleware 验证 JWT 的中间件
// 从 Authorization: Bearer <token> 中提取 Token 并校验
// 校验通过后将用户信息写入 Context
func AuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "缺少 Authorization 头"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization 头格式错误"})
			return
		}

		claims, err := utils.ParseToken(parts[1], secret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "无效或过期的 Token"})
			return
		}

		// 将用户信息写入 Context，供后续 handler 使用
		// 注意：存入 uint，与 handler 中的 c.GetUint 精确断言匹配
		c.Set("user_id", uint(claims.UserID))
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}
