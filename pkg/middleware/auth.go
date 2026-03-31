package middleware

import (
	"cmdb-v2/pkg/common"
	"cmdb-v2/pkg/utils"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

// AuthMiddleware 统一鉴权中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 获取 Authorization Header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			common.Error(c, http.StatusUnauthorized, "未提供认证信息")
			c.Abort()
			return
		}

		// 2. 解析 Bearer Token
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			common.Error(c, http.StatusUnauthorized, "认证格式错误")
			c.Abort()
			return
		}

		// 3. 验证 Token (调用 pkg/utils/jwt.go)
		claims, err := utils.ParseToken(parts[1])
		if err != nil {
			common.Error(c, http.StatusUnauthorized, "无效的 Token 或 Token 已过期")
			c.Abort()
			return
		}

		// 4. 将用户信息存入上下文，方便后续 Handler 获取
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("permissions", claims.Permissions)

		c.Next()
	}
}
