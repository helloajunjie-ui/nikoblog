package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"nikoblog/internal/utils"
)

// OptionalAuthMiddleware 可选鉴权中间件
// 用于公开的查询接口：若请求携带有效 Token 则设置 user_id（登录用户），
// 否则不拦截（视为未登录用户）。绝不因 Token 缺失或无效而拒绝请求。
func OptionalAuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.Next()
			return
		}

		claims, err := utils.ParseToken(parts[1], secret)
		if err != nil {
			// Token 无效：视为未登录，不拦截
			c.Next()
			return
		}

		// 设置用户信息，供 handler 做可见性过滤
		// 注意：存入 uint，与 handler 中的 c.GetUint 精确断言匹配
		c.Set("user_id", uint(claims.UserID))
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}
