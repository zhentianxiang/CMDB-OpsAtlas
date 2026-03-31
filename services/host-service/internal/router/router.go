package router

import (
	"cmdb-v2/pkg/middleware"
	"cmdb-v2/services/host-service/internal/handlers"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Init(r *gin.Engine, h *handlers.Handler, db *gorm.DB) {
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1/hosts")
	api.Use(middleware.AuthMiddleware(), middleware.AuditMiddleware(db))
	{
		api.GET("", middleware.RequirePermission("cmdb:view"), h.List)
		api.POST("", middleware.RequirePermission("cmdb:host:create"), h.Create)
		api.GET("/:id", middleware.RequirePermission("cmdb:view"), h.Get)
		api.GET("/:id/detail", middleware.RequirePermission("cmdb:view"), h.GetDetail)
		api.PUT("/:id", middleware.RequirePermission("cmdb:host:update"), h.Update)
		api.DELETE("/:id", middleware.RequirePermission("cmdb:host:delete"), h.Delete)
	}
}
