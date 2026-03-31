package router

import (
	"cmdb-v2/pkg/middleware"
	"cmdb-v2/services/auth-service/internal/handlers"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func InitRouter(r *gin.Engine, authHandler *handlers.AuthHandler, systemHandler *handlers.SystemHandler, db *gorm.DB) {
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1/auth")
	{
		api.POST("/login", authHandler.Login)
	}

	protected := api.Group("")
	// 使用全局统一的 Auth 和 Audit 中间件
	protected.Use(middleware.AuthMiddleware(), middleware.AuditMiddleware(db))
	{
		protected.GET("/me", authHandler.GetMine)
		protected.PUT("/me", authHandler.UpdateMine)
		protected.POST("/me/avatar", authHandler.UploadMyAvatar)
		protected.PUT("/password", authHandler.UpdateMyPassword)

		// 用户管理接口（管理员）
		protected.GET("/users", authHandler.ListUsers)
		protected.POST("/users", authHandler.CreateUser)
		protected.PUT("/users/:id", authHandler.UpdateUser)
		protected.POST("/users/:id/avatar", authHandler.UploadUserAvatar)
		protected.PUT("/users/:id/password", authHandler.ResetPassword) // 新增此行
		protected.PUT("/users/:id/role", authHandler.UpdateUserRole)
		protected.DELETE("/users/:id", authHandler.DeleteUser)
		protected.POST("/user-role-ids", authHandler.GetUserRoleIDs)

		// 系统管理接口
		protected.GET("/dept", systemHandler.ListDepts)
		protected.POST("/dept", systemHandler.CreateDept)
		protected.PUT("/dept/:id", systemHandler.UpdateDept)
		protected.DELETE("/dept/:id", systemHandler.DeleteDept)

		// 审计日志接口
		protected.GET("/audit-logs", systemHandler.ListAuditLogs)

		// 角色管理
		protected.POST("/role", systemHandler.ListRoles)          // 下拉框获取或全量列表
		protected.POST("/roles", systemHandler.ListRolesForTable) // 角色管理分页列表
		protected.POST("/role/create", systemHandler.CreateRole)
		protected.PUT("/role/:id", systemHandler.UpdateRole)
		protected.DELETE("/role/:id", systemHandler.DeleteRole)

		// 菜单管理
		protected.POST("/menu", systemHandler.ListMenus)
		protected.POST("/menu/create", systemHandler.CreateMenu)
		protected.PUT("/menu/:id", systemHandler.UpdateMenu)
		protected.DELETE("/menu/:id", systemHandler.DeleteMenu)

		// 角色-菜单关联接口
		protected.POST("/role-menu", systemHandler.ListMenus)
		protected.POST("/role-menu-ids", systemHandler.GetRoleMenuIds)
		protected.POST("/list-role-ids", authHandler.GetUserRoleIDs) // 兼容用户角色分配弹窗
		protected.POST("/update-role-menus", systemHandler.UpdateRoleMenus)
	}
}
