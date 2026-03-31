package router

import (
	"cmdb-v2/pkg/middleware"
	"cmdb-v2/services/topology-service/internal/handlers"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Init(r *gin.Engine, h *handlers.Handler) {
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	api := r.Group("/api/v1/topology")
	api.Use(middleware.AuthMiddleware())
	{
		api.GET("", middleware.RequirePermission("cmdb:view"), h.GetTopology)
	}
}
