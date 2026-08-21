package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"nikoblog/internal/models"
)

// AdminMiddleware 校验当前用户是否为 admin 角色
// 必须放在 AuthMiddleware 之后使用（依赖 context 中的 role）
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("role")
		if role != models.RoleAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "无管理员权限"})
			return
		}
		c.Next()
	}
}
