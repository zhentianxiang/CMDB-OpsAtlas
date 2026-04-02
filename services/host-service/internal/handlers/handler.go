package handlers

import (
	"cmdb-v2/pkg/common"
	"cmdb-v2/pkg/models"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct{ DB *gorm.DB }

func New(db *gorm.DB) *Handler { return &Handler{DB: db} }

type CreateHostRequest struct {
	Name      string `json:"name" binding:"required"`
	IP        string `json:"ip"`
	PublicIP  string `json:"public_ip"`
	PrivateIP string `json:"private_ip"`
	ClusterID *uint  `json:"cluster_id"`
	CPU       int    `json:"cpu"`
	Memory    int    `json:"memory"`
	OS        string `json:"os"`
	Status    string `json:"status"`
	Remark    string `json:"remark"`
}

type appWithPorts struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Ports []int  `json:"ports"`
}

type dependencyView struct {
	SourceAppID  *uint  `json:"source_app_id"`
	TargetAppID  *uint  `json:"target_app_id"`
	SourceHostID *uint  `json:"source_host_id"`
	TargetHostID *uint  `json:"target_host_id"`
	SourceNode   string `json:"source_node"`
	TargetNode   string `json:"target_node"`
	Desc         string `json:"desc"`
	Remark       string `json:"remark"`
}

type hostDetailResponse struct {
	Host          gin.H            `json:"host"`
	Cluster       *models.Cluster  `json:"cluster,omitempty"`
	Apps          []appWithPorts   `json:"apps"`
	Domains       []string         `json:"domains"`
	CallsOutgoing []dependencyView `json:"calls_outgoing"`
	CallsIncoming []dependencyView `json:"calls_incoming"`
}

func (h *Handler) List(c *gin.Context) {
	var hosts []models.Host
	q := h.DB.Model(&models.Host{}).
		Joins("LEFT JOIN clusters ON clusters.id = hosts.cluster_id AND clusters.deleted_at IS NULL")

	// 数据隔离
	q = h.filterByUser(c, q, "hosts")

	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		like := "%" + strings.ToLower(keyword) + "%"
		q = q.Where(`
			LOWER(hosts.name) LIKE ? OR
			LOWER(COALESCE(hosts.ip, '')) LIKE ? OR
			LOWER(COALESCE(hosts.public_ip, '')) LIKE ? OR
			LOWER(COALESCE(hosts.private_ip, '')) LIKE ? OR
			LOWER(COALESCE(hosts.status, '')) LIKE ? OR
			LOWER(COALESCE(hosts.os, '')) LIKE ? OR
			LOWER(COALESCE(hosts.remark, '')) LIKE ? OR
			LOWER(COALESCE(clusters.name, '')) LIKE ?
		`, like, like, like, like, like, like, like, like)
	}

	if err := q.Distinct("hosts.*").Order("hosts.id ASC").Find(&hosts).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "获取主机列表失败: "+err.Error())
		return
	}
	common.Success(c, hosts)
}

func (h *Handler) Get(c *gin.Context) {
	var host models.Host
	q := h.DB.Model(&models.Host{})
	q = h.filterByUser(c, q, "")

	if err := q.First(&host, c.Param("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			common.Error(c, http.StatusNotFound, "主机不存在")
			return
		}
		common.Error(c, http.StatusInternalServerError, "获取主机详情失败: "+err.Error())
		return
	}
	common.Success(c, host)
}

func (h *Handler) GetDetail(c *gin.Context) {
	hostID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Error(c, http.StatusBadRequest, "无效的主机ID")
		return
	}

	var host models.Host
	q := h.DB.Model(&models.Host{})
	q = h.filterByUser(c, q, "")

	if err := q.First(&host, hostID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			common.Error(c, http.StatusNotFound, "主机不存在")
			return
		}
		common.Error(c, http.StatusInternalServerError, "获取主机失败: "+err.Error())
		return
	}

	var cluster *models.Cluster
	if host.ClusterID != nil {
		var citem models.Cluster
		if err := h.DB.First(&citem, *host.ClusterID).Error; err == nil {
			cluster = &citem
		}
	}

	var appsRaw []models.App
	if err := h.DB.Where("host_id = ?", host.ID).Find(&appsRaw).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "查询应用失败: "+err.Error())
		return
	}

	appIDs := make([]uint, 0, len(appsRaw))
	for _, app := range appsRaw {
		appIDs = append(appIDs, app.ID)
	}

	var ports []models.Port
	if len(appIDs) > 0 {
		if err := h.DB.Where("app_id IN ?", appIDs).Find(&ports).Error; err != nil {
			common.Error(c, http.StatusInternalServerError, "查询端口失败: "+err.Error())
			return
		}
	}
	portMap := make(map[uint][]int)
	for _, p := range ports {
		portMap[p.AppID] = append(portMap[p.AppID], p.Port)
	}

	apps := make([]appWithPorts, 0, len(appsRaw))
	for _, app := range appsRaw {
		apps = append(apps, appWithPorts{ID: app.ID, Name: app.Name, Ports: portMap[app.ID]})
	}

	var domains []models.Domain
	q = h.DB.Where("host_id = ?", host.ID)
	if len(appIDs) > 0 {
		q = q.Or("app_id IN ?", appIDs)
	}
	if err := q.Find(&domains).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "查询域名失败: "+err.Error())
		return
	}

	domainSet := make(map[string]struct{})
	domainNames := make([]string, 0, len(domains))
	for _, d := range domains {
		if _, exists := domainSet[d.Domain]; exists {
			continue
		}
		domainSet[d.Domain] = struct{}{}
		domainNames = append(domainNames, d.Domain)
	}

	var outgoingDeps []models.Dependency
	if err := h.DB.Where("source_host_id = ?", host.ID).Find(&outgoingDeps).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "查询调用关系失败: "+err.Error())
		return
	}

	var incomingDeps []models.Dependency
	if err := h.DB.Where("target_host_id = ?", host.ID).Find(&incomingDeps).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "查询被调用关系失败: "+err.Error())
		return
	}

	convertDeps := func(input []models.Dependency) []dependencyView {
		out := make([]dependencyView, 0, len(input))
		for _, item := range input {
			out = append(out, dependencyView{
				SourceAppID:  item.SourceAppID,
				TargetAppID:  item.TargetAppID,
				SourceHostID: item.SourceHostID,
				TargetHostID: item.TargetHostID,
				SourceNode:   item.SourceNode,
				TargetNode:   item.TargetNode,
				Desc:         item.Desc,
				Remark:       item.Remark,
			})
		}
		return out
	}

	ip := host.PrivateIP
	if ip == "" {
		if host.IP != "" {
			ip = host.IP
		} else {
			ip = host.PublicIP
		}
	}

	common.Success(c, hostDetailResponse{
		Host:          gin.H{"id": host.ID, "name": host.Name, "ip": ip},
		Cluster:       cluster,
		Apps:          apps,
		Domains:       domainNames,
		CallsOutgoing: convertDeps(outgoingDeps),
		CallsIncoming: convertDeps(incomingDeps),
	})
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateHostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	userID, _ := c.Get("user_id")
	uID, _ := userID.(uint)

	status := req.Status
	if status == "" {
		status = "online"
	}
	host := models.Host{
		UserID:    uID,
		Name:      req.Name,
		IP:        req.IP,
		PublicIP:  req.PublicIP,
		PrivateIP: req.PrivateIP,
		ClusterID: req.ClusterID,
		CPU:       req.CPU,
		Memory:    req.Memory,
		OS:        req.OS,
		Status:    status,
	}
	host.Remark = req.Remark
	if host.IP == "" {
		host.IP = host.PrivateIP
	}
	if err := h.DB.Create(&host).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}
	common.Success(c, host)
}

func (h *Handler) Update(c *gin.Context) {
	var host models.Host
	q := h.DB.Model(&models.Host{})
	q = h.filterByUser(c, q, "")

	if err := q.First(&host, c.Param("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			common.Error(c, http.StatusNotFound, "主机不存在")
			return
		}
		common.Error(c, http.StatusInternalServerError, "查询主机失败: "+err.Error())
		return
	}
	var req CreateHostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	host.Name, host.IP, host.PublicIP, host.PrivateIP = req.Name, req.IP, req.PublicIP, req.PrivateIP
	host.ClusterID, host.CPU, host.Memory, host.OS, host.Status = req.ClusterID, req.CPU, req.Memory, req.OS, req.Status
	host.Remark = req.Remark
	if host.IP == "" {
		host.IP = host.PrivateIP
	}
	if err := h.DB.Save(&host).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "更新主机失败: "+err.Error())
		return
	}
	common.Success(c, host)
}

func (h *Handler) Delete(c *gin.Context) {
	var host models.Host
	q := h.DB.Model(&models.Host{})
	q = h.filterByUser(c, q, "")

	if err := q.First(&host, c.Param("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			common.Error(c, http.StatusNotFound, "主机不存在")
			return
		}
		common.Error(c, http.StatusInternalServerError, "查询主机失败: "+err.Error())
		return
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		var appIDs []uint
		if err := tx.Model(&models.App{}).
			Where("host_id = ?", host.ID).
			Pluck("id", &appIDs).Error; err != nil {
			return err
		}

		// 删除主机时一并清理依赖它的应用、端口、域名和依赖关系，避免留下孤儿数据。
		if len(appIDs) > 0 {
			if err := tx.Unscoped().Where("app_id IN ?", appIDs).Delete(&models.Port{}).Error; err != nil {
				return err
			}
			if err := tx.Unscoped().
				Where("app_id IN ?", appIDs).
				Or("host_id = ?", host.ID).
				Delete(&models.Domain{}).Error; err != nil {
				return err
			}
			if err := tx.Unscoped().
				Where("source_app_id IN ?", appIDs).
				Or("target_app_id IN ?", appIDs).
				Or("source_host_id = ?", host.ID).
				Or("target_host_id = ?", host.ID).
				Delete(&models.Dependency{}).Error; err != nil {
				return err
			}
			if err := tx.Unscoped().Where("id IN ?", appIDs).Delete(&models.App{}).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Unscoped().
				Where("host_id = ?", host.ID).
				Delete(&models.Domain{}).Error; err != nil {
				return err
			}
			if err := tx.Unscoped().
				Where("source_host_id = ?", host.ID).
				Or("target_host_id = ?", host.ID).
				Delete(&models.Dependency{}).Error; err != nil {
				return err
			}
		}

		return tx.Unscoped().Delete(&host).Error
	}); err != nil {
		common.Error(c, http.StatusInternalServerError, "删除失败: "+err.Error())
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
