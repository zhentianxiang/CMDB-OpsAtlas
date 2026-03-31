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
	Domain string `json:"domain" binding:"required"`
	AppID  *uint  `json:"app_id"`
	HostID *uint  `json:"host_id"`
	Remark string `json:"remark"`
}

func (h *Handler) List(c *gin.Context) {
	var items []models.Domain
	q := h.DB.Model(&models.Domain{}).
		Joins("LEFT JOIN apps ON apps.id = domains.app_id AND apps.deleted_at IS NULL").
		Joins("LEFT JOIN hosts ON hosts.id = domains.host_id AND hosts.deleted_at IS NULL")

	// 数据隔离
	q = h.filterByUser(c, q, "domains")


	if appID := c.Query("app_id"); appID != "" {
		q = q.Where("domains.app_id = ?", appID)
	}
	if hostID := c.Query("host_id"); hostID != "" {
		q = q.Where("domains.host_id = ?", hostID)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		like := "%" + strings.ToLower(keyword) + "%"
		q = q.Where(`
			(LOWER(domains.domain) LIKE ? OR
			LOWER(COALESCE(domains.remark, '')) LIKE ? OR
			LOWER(COALESCE(apps.name, '')) LIKE ? OR
			LOWER(COALESCE(hosts.name, '')) LIKE ?)
		`, like, like, like, like)
	}
	if err := q.Distinct("domains.*").Order("domains.id ASC").Find(&items).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "查询域名失败: "+err.Error())
		return
	}
	common.Success(c, items)
}

func (h *Handler) Get(c *gin.Context) {
	var item models.Domain
	q := h.DB.Model(&models.Domain{})
	q = h.filterByUser(c, q, "")

	if err := q.First(&item, c.Param("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			common.Error(c, http.StatusNotFound, "域名不存在")
			return
		}
		common.Error(c, http.StatusInternalServerError, "查询域名失败: "+err.Error())
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

	item := models.Domain{
		UserID: uID,
		Domain: req.Domain,
		AppID:  req.AppID,
		HostID: req.HostID,
		Remark: req.Remark,
	}
	if err := h.DB.Create(&item).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "创建域名失败: "+err.Error())
		return
	}
	common.Success(c, item)
}

func (h *Handler) Update(c *gin.Context) {
	var item models.Domain
	q := h.DB.Model(&models.Domain{})
	q = h.filterByUser(c, q, "")

	if err := q.First(&item, c.Param("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			common.Error(c, http.StatusNotFound, "域名不存在")
			return
		}
		common.Error(c, http.StatusInternalServerError, "查询域名失败: "+err.Error())
		return
	}
	var req Request
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	item.Domain, item.AppID, item.HostID = req.Domain, req.AppID, req.HostID
	item.Remark = req.Remark
	if err := h.DB.Save(&item).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "更新域名失败: "+err.Error())
		return
	}
	common.Success(c, item)
}

func (h *Handler) Delete(c *gin.Context) {
	var item models.Domain
	q := h.DB.Model(&models.Domain{})
	q = h.filterByUser(c, q, "")

	if err := q.First(&item, c.Param("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			common.Error(c, http.StatusNotFound, "域名不存在")
			return
		}
		common.Error(c, http.StatusInternalServerError, "查询域名失败: "+err.Error())
		return
	}

	if err := h.DB.Unscoped().Delete(&item).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "删除域名失败: "+err.Error())
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
