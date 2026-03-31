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
	SourceAppID  *uint  `json:"source_app_id"`
	TargetAppID  *uint  `json:"target_app_id"`
	SourceHostID *uint  `json:"source_host_id"`
	TargetHostID *uint  `json:"target_host_id"`
	SourceNode   string `json:"source_node"`
	TargetNode   string `json:"target_node"`
	DomainID     *uint  `json:"domain_id"`
	Desc         string `json:"desc"`
	Remark       string `json:"remark"`
}

func (h *Handler) List(c *gin.Context) {
	var items []models.Dependency
	q := h.DB.Model(&models.Dependency{}).
		Joins("LEFT JOIN apps AS source_apps ON source_apps.id = dependencies.source_app_id AND source_apps.deleted_at IS NULL").
		Joins("LEFT JOIN apps AS target_apps ON target_apps.id = dependencies.target_app_id AND target_apps.deleted_at IS NULL").
		Joins("LEFT JOIN hosts AS source_hosts ON source_hosts.id = dependencies.source_host_id AND source_hosts.deleted_at IS NULL").
		Joins("LEFT JOIN hosts AS target_hosts ON target_hosts.id = dependencies.target_host_id AND target_hosts.deleted_at IS NULL")

	// 数据隔离
	q = h.filterByUser(c, q, "dependencies")


	if sourceAppID := c.Query("source_app_id"); sourceAppID != "" {
		q = q.Where("dependencies.source_app_id = ?", sourceAppID)
	}
	if targetAppID := c.Query("target_app_id"); targetAppID != "" {
		q = q.Where("dependencies.target_app_id = ?", targetAppID)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		like := "%" + strings.ToLower(keyword) + "%"
		q = q.Where(`
			(LOWER(COALESCE(dependencies.source_node, '')) LIKE ? OR
			LOWER(COALESCE(dependencies.target_node, '')) LIKE ? OR
			LOWER(COALESCE(dependencies.desc, '')) LIKE ? OR
			LOWER(COALESCE(dependencies.remark, '')) LIKE ? OR
			LOWER(COALESCE(source_apps.name, '')) LIKE ? OR
			LOWER(COALESCE(target_apps.name, '')) LIKE ? OR
			LOWER(COALESCE(source_hosts.name, '')) LIKE ? OR
			LOWER(COALESCE(target_hosts.name, '')) LIKE ?)
		`, like, like, like, like, like, like, like, like)
	}
	if err := q.Distinct("dependencies.*").Order("dependencies.id ASC").Find(&items).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "查询依赖失败: "+err.Error())
		return
	}
	common.Success(c, items)
}

func (h *Handler) Get(c *gin.Context) {
	var item models.Dependency
	q := h.DB.Model(&models.Dependency{})
	q = h.filterByUser(c, q, "")

	if err := q.First(&item, c.Param("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			common.Error(c, http.StatusNotFound, "依赖关系不存在")
			return
		}
		common.Error(c, http.StatusInternalServerError, "查询依赖失败: "+err.Error())
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
	if (req.SourceAppID == nil && req.SourceHostID == nil && req.SourceNode == "") ||
		(req.TargetAppID == nil && req.TargetHostID == nil && req.TargetNode == "") {
		common.Error(c, http.StatusBadRequest, "source 和 target 至少要提供 app_id/host_id/node 之一")
		return
	}

	userID, _ := c.Get("user_id")
	uID, _ := userID.(uint)

	item := models.Dependency{
		UserID:       uID,
		SourceAppID:  req.SourceAppID,
		TargetAppID:  req.TargetAppID,
		SourceHostID: req.SourceHostID,
		TargetHostID: req.TargetHostID,
		SourceNode:   req.SourceNode,
		TargetNode:   req.TargetNode,
		DomainID:     req.DomainID,
		Desc:         req.Desc,
		Remark:       req.Remark,
	}
	if err := h.DB.Create(&item).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "创建依赖失败: "+err.Error())
		return
	}
	common.Success(c, item)
}

func (h *Handler) Update(c *gin.Context) {
	var item models.Dependency
	q := h.DB.Model(&models.Dependency{})
	q = h.filterByUser(c, q, "")

	if err := q.First(&item, c.Param("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			common.Error(c, http.StatusNotFound, "依赖关系不存在")
			return
		}
		common.Error(c, http.StatusInternalServerError, "查询依赖失败: "+err.Error())
		return
	}
	var req Request
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if (req.SourceAppID == nil && req.SourceHostID == nil && req.SourceNode == "") ||
		(req.TargetAppID == nil && req.TargetHostID == nil && req.TargetNode == "") {
		common.Error(c, http.StatusBadRequest, "source 和 target 至少要提供 app_id/host_id/node 之一")
		return
	}

	item.SourceAppID, item.TargetAppID = req.SourceAppID, req.TargetAppID
	item.SourceHostID, item.TargetHostID = req.SourceHostID, req.TargetHostID
	item.SourceNode, item.TargetNode = req.SourceNode, req.TargetNode
	item.DomainID = req.DomainID
	item.Desc = req.Desc
	item.Remark = req.Remark
	if err := h.DB.Save(&item).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "更新依赖失败: "+err.Error())
		return
	}
	common.Success(c, item)
}

func (h *Handler) Delete(c *gin.Context) {
	var item models.Dependency
	q := h.DB.Model(&models.Dependency{})
	q = h.filterByUser(c, q, "")

	if err := q.First(&item, c.Param("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			common.Error(c, http.StatusNotFound, "依赖关系不存在")
			return
		}
		common.Error(c, http.StatusInternalServerError, "查询依赖失败: "+err.Error())
		return
	}

	if err := h.DB.Unscoped().Delete(&item).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "删除依赖失败: "+err.Error())
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
