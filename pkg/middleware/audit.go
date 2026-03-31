package middleware

import (
	"bytes"
	"cmdb-v2/pkg/models"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func AuditMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 只审计修改类操作
		if c.Request.Method == "GET" {
			c.Next()
			return
		}

		start := time.Now()

		// 2. 读取 payload (注意：这里需要备份 body 供后续逻辑读取)
		var body []byte
		if c.Request.Body != nil {
			body, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		}

		// 执行后续逻辑
		c.Next()

		// 3. 记录日志 (在请求结束后执行)
		userID, _ := c.Get("user_id")
		username, _ := c.Get("username")
		uid, _ := userID.(uint)
		uname, _ := username.(string)

		auditLog := models.AuditLog{
			UserID:    uid,
			Username:  uname,
			Method:    c.Request.Method,
			Path:      c.Request.URL.Path,
			IP:        c.ClientIP(),
			Status:    c.Writer.Status(),
			Duration:  time.Since(start).Milliseconds(),
			Payload:   string(body),
			Operation: translatePathToOp(c.Request.Method, c.Request.URL.Path),
		}

		// 异步或直接写入数据库 (这里直接写入确保一致性)
		db.Create(&auditLog)
	}
}

// translatePathToOp 将 API 路径翻译为更详细的操作描述
func translatePathToOp(method, path string) string {
	action := ""
	switch method {
	case "POST":
		action = "创建"
	case "PUT":
		action = "修改"
	case "DELETE":
		action = "删除"
	default:
		action = "操作"
	}

	resource := ""
	p := strings.ToLower(path)
	
	// 匹配资源类型
	if strings.Contains(p, "/auth/users") {
		resource = "用户"
	} else if strings.Contains(p, "/auth/role") {
		resource = "角色"
	} else if strings.Contains(p, "/auth/menu") {
		resource = "菜单"
	} else if strings.Contains(p, "/auth/dept") {
		resource = "部门"
	} else if strings.Contains(p, "/clusters") {
		resource = "集群"
	} else if strings.Contains(p, "/hosts") {
		resource = "主机"
	} else if strings.Contains(p, "/apps") {
		resource = "应用"
	} else if strings.Contains(p, "/ports") {
		resource = "端口"
	} else if strings.Contains(p, "/domains") {
		resource = "域名"
	} else if strings.Contains(p, "/dependencies") {
		resource = "依赖关系"
	} else if strings.Contains(p, "/backup") {
		resource = "备份策略/文件"
	} else if strings.Contains(p, "/transfer") {
		resource = "数据导入导出"
	} else {
		resource = "资源: " + path
	}

	return action + resource
}
