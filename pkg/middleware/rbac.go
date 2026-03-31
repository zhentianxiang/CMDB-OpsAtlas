package middleware

import (
	"cmdb-v2/pkg/common"
	"net/http"

	"github.com/gin-gonic/gin"
)

const wildcardPermission = "*:*:*"

func getPermissionSet(c *gin.Context) map[string]struct{} {
	raw, exists := c.Get("permissions")
	if !exists {
		return map[string]struct{}{}
	}

	permissions, ok := raw.([]string)
	if !ok {
		return map[string]struct{}{}
	}

	set := make(map[string]struct{}, len(permissions))
	for _, item := range permissions {
		if item == "" {
			continue
		}
		set[item] = struct{}{}
	}
	return set
}

func HasPermission(c *gin.Context, permission string) bool {
	if permission == "" {
		return true
	}

	if role, exists := c.Get("role"); exists && role == "admin" {
		return true
	}

	set := getPermissionSet(c)
	if _, ok := set[wildcardPermission]; ok {
		return true
	}
	_, ok := set[permission]
	return ok
}

func HasAnyPermission(c *gin.Context, permissions ...string) bool {
	if len(permissions) == 0 {
		return true
	}
	for _, permission := range permissions {
		if HasPermission(c, permission) {
			return true
		}
	}
	return false
}

func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if HasPermission(c, permission) {
			c.Next()
			return
		}
		common.Error(c, http.StatusForbidden, "当前用户没有访问该资源的权限")
		c.Abort()
	}
}

func RequireAnyPermission(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if HasAnyPermission(c, permissions...) {
			c.Next()
			return
		}
		common.Error(c, http.StatusForbidden, "当前用户没有访问该资源的权限")
		c.Abort()
	}
}
