package router

import (
	"cmdb-v2/pkg/middleware"
	"cmdb-v2/services/cmdb-transfer-service/internal/handlers"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Init(r *gin.Engine, h *handlers.Handler, db *gorm.DB) {
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(), middleware.AuditMiddleware(db))
	{
		// 1. 操作审计与概览 (ops)
		ops := api.Group("/ops")
		{
			ops.GET("/overview", middleware.RequirePermission("ops:runtime:view"), h.GetOpsOverview)
			ops.GET("/transfer-records", middleware.RequireAnyPermission("ops:task:view", "ops:logs:view", "ops:backup:view"), h.ListTransferRecords)
			ops.GET("/service-logs", middleware.RequirePermission("ops:logs:view"), h.GetServiceLogs)
			ops.POST("/sql/query", middleware.RequirePermission("ops:sql:read"), h.QuerySQL)
			ops.POST("/sql/execute", middleware.RequirePermission("ops:sql:execute"), h.ExecuteSQL)
		}

		// 2. 资产导入导出 (前端匹配修正)
		transfer := api.Group("/cmdb")
		{
			transfer.GET("/export", middleware.RequirePermission("cmdb:export"), h.ExportCMDB)
			transfer.POST("/import", middleware.RequirePermission("cmdb:import"), h.ImportCMDB)
			transfer.POST("/import/preview", middleware.RequirePermission("cmdb:import"), h.PreviewImportCMDB)
			transfer.POST("/template", middleware.RequirePermission("cmdb:import"), h.DownloadTemplate)
		}

		// 3. 备份管理 (backup)
		backup := api.Group("/backup")
		{
			backup.GET("/policy", middleware.RequirePermission("ops:backup:view"), h.GetBackupPolicy)
			backup.PUT("/policy", middleware.RequirePermission("ops:backup:manage"), h.UpdateBackupPolicy)
			backup.GET("/files", middleware.RequirePermission("ops:backup:view"), h.ListBackupFiles)
			backup.POST("/run", middleware.RequirePermission("ops:backup:manage"), h.RunBackupNow)
			backup.GET("/files/:id/download", middleware.RequirePermission("ops:backup:view"), h.DownloadBackupFile)
			backup.POST("/files/:id/restore", middleware.RequirePermission("ops:backup:manage"), h.RestoreBackupFile)
			backup.DELETE("/files/:id", middleware.RequirePermission("ops:backup:manage"), h.DeleteBackupFile)
		}
	}
}
