package handlers

import (
	"cmdb-v2/pkg/common"
	"cmdb-v2/pkg/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct{ DB *gorm.DB }

func New(db *gorm.DB) *Handler { return &Handler{DB: db} }

type Request struct {
	Name       string `json:"name" binding:"required"`
	HostID     uint   `json:"host_id" binding:"required"`
	Type       string `json:"type"`
	Version    string `json:"version"`
	DeployType string `json:"deploy_type"`
	Remark     string `json:"remark"`
}

func (h *Handler) List(c *gin.Context) {
	var items []models.App
	q := h.DB.Model(&models.App{}).
		Joins("LEFT JOIN hosts ON hosts.id = apps.host_id AND hosts.deleted_at IS NULL")

	// 数据隔离
	q = h.filterByUser(c, q, "apps")


	if hostID := c.Query("host_id"); hostID != "" {
		q = q.Where("apps.host_id = ?", hostID)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		like := "%" + strings.ToLower(keyword) + "%"
		q = q.Where(`
			(LOWER(apps.name) LIKE ? OR
			LOWER(COALESCE(apps.type, '')) LIKE ? OR
			LOWER(COALESCE(apps.version, '')) LIKE ? OR
			LOWER(COALESCE(apps.deploy_type, '')) LIKE ? OR
			LOWER(COALESCE(apps.remark, '')) LIKE ? OR
			LOWER(COALESCE(hosts.name, '')) LIKE ?)
		`, like, like, like, like, like, like)
	}
	if err := q.Distinct("apps.*").Order("apps.id ASC").Find(&items).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "查询应用失败: "+err.Error())
		return
	}
	common.Success(c, items)
}

func (h *Handler) Get(c *gin.Context) {
	var item models.App
	q := h.DB.Model(&models.App{})
	q = h.filterByUser(c, q, "")

	if err := q.First(&item, c.Param("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			common.Error(c, http.StatusNotFound, "应用不存在")
			return
		}
		common.Error(c, http.StatusInternalServerError, "查询应用失败: "+err.Error())
		return
	}
	common.Success(c, item)
}

func (h *Handler) Create(c *gin.Context) {
	var req Request
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	userID, _ := c.Get("user_id")
	uID, _ := userID.(uint)

	item := models.App{
		UserID:     uID,
		Name:       req.Name,
		HostID:     req.HostID,
		Type:       req.Type,
		Version:    req.Version,
		DeployType: req.DeployType,
		Remark:     req.Remark,
	}
	if err := h.DB.Create(&item).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "创建应用失败: "+err.Error())
		return
	}
	common.Success(c, item)
}

func (h *Handler) Update(c *gin.Context) {
	var item models.App
	q := h.DB.Model(&models.App{})
	q = h.filterByUser(c, q, "")

	if err := q.First(&item, c.Param("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			common.Error(c, http.StatusNotFound, "应用不存在")
			return
		}
		common.Error(c, http.StatusInternalServerError, "查询应用失败: "+err.Error())
		return
	}
	var req Request
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	item.Name, item.HostID, item.Type, item.Version, item.DeployType = req.Name, req.HostID, req.Type, req.Version, req.DeployType
	item.Remark = req.Remark
	if err := h.DB.Save(&item).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "更新应用失败: "+err.Error())
		return
	}
	common.Success(c, item)
}

func (h *Handler) Delete(c *gin.Context) {
	var item models.App
	q := h.DB.Model(&models.App{})
	q = h.filterByUser(c, q, "")

	if err := q.First(&item, c.Param("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			common.Error(c, http.StatusNotFound, "应用不存在")
			return
		}
		common.Error(c, http.StatusInternalServerError, "查询应用失败: "+err.Error())
		return
	}

	if err := h.DB.Unscoped().Delete(&item).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "删除应用失败: "+err.Error())
		return
	}
	common.Success(c, nil)
}

func (h *Handler) filterByUser(c *gin.Context, q *gorm.DB, table string) *gorm.DB {
	role, _ := c.Get("role")
	if role == "admin" {
		return q
	}

	userID, exists := c.Get("user_id")
	if !exists {
		return q.Where("1 = 0")
	}

	column := "user_id"
	if table != "" {
		column = table + ".user_id"
	}

	// 转换为 uint
	var uID uint
	switch v := userID.(type) {
	case uint:
		uID = v
	case float64:
		uID = uint(v)
	case int:
		uID = uint(v)
	}

	return q.Where(column+" = ?", uID)
}
