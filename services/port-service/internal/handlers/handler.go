package handlers

import (
	"cmdb-v2/pkg/common"
	"cmdb-v2/pkg/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct{ DB *gorm.DB }

func New(db *gorm.DB) *Handler { return &Handler{DB: db} }

type Request struct {
	AppID    uint   `json:"app_id" binding:"required"`
	Port     int    `json:"port" binding:"required"`
	Protocol string `json:"protocol"`
	IsPublic bool   `json:"is_public"`
	Remark   string `json:"remark"`
}

func (h *Handler) List(c *gin.Context) {
	var items []models.Port
	q := h.DB.Model(&models.Port{})
	q = h.filterByUser(c, q)

	if appID := c.Query("app_id"); appID != "" {

		q = q.Where("app_id = ?", appID)
	}
	if err := q.Find(&items).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "查询端口失败: "+err.Error())
		return
	}
	common.Success(c, items)
}

func (h *Handler) Get(c *gin.Context) {
	var item models.Port
	q := h.DB.Model(&models.Port{})
	q = h.filterByUser(c, q)

	if err := q.First(&item, c.Param("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			common.Error(c, http.StatusNotFound, "端口不存在")
			return
		}
		common.Error(c, http.StatusInternalServerError, "查询端口失败: "+err.Error())
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

	item := models.Port{
		UserID:   uID,
		AppID:    req.AppID,
		Port:     req.Port,
		Protocol: req.Protocol,
		IsPublic: req.IsPublic,
		Remark:   req.Remark,
	}
	if err := h.DB.Create(&item).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "创建端口失败: "+err.Error())
		return
	}
	common.Success(c, item)
}

func (h *Handler) Update(c *gin.Context) {
	var item models.Port
	q := h.DB.Model(&models.Port{})
	q = h.filterByUser(c, q)

	if err := q.First(&item, c.Param("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			common.Error(c, http.StatusNotFound, "端口不存在")
			return
		}
		common.Error(c, http.StatusInternalServerError, "查询端口失败: "+err.Error())
		return
	}
	var req Request
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	item.AppID, item.Port, item.Protocol, item.IsPublic = req.AppID, req.Port, req.Protocol, req.IsPublic
	item.Remark = req.Remark
	if err := h.DB.Save(&item).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "更新端口失败: "+err.Error())
		return
	}
	common.Success(c, item)
}

func (h *Handler) Delete(c *gin.Context) {
	var item models.Port
	q := h.DB.Model(&models.Port{})
	q = h.filterByUser(c, q)

	if err := q.First(&item, c.Param("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			common.Error(c, http.StatusNotFound, "端口不存在")
			return
		}
		common.Error(c, http.StatusInternalServerError, "查询端口失败: "+err.Error())
		return
	}

	if err := h.DB.Unscoped().Delete(&item).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "删除端口失败: "+err.Error())
		return
	}
	common.Success(c, nil)
}

func (h *Handler) filterByUser(c *gin.Context, q *gorm.DB) *gorm.DB {
	role, _ := c.Get("role")
	if role == "admin" {
		return q
	}

	userID, exists := c.Get("user_id")
	if !exists {
		return q.Where("1 = 0")
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

	return q.Where("user_id = ?", uID)
}
