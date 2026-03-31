package handlers

import (
	"cmdb-v2/pkg/common"
	"cmdb-v2/pkg/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct{ DB *gorm.DB }

func New(db *gorm.DB) *Handler { return &Handler{DB: db} }

type topologyNode struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type topologyLink struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

func (h *Handler) GetTopology(c *gin.Context) {
	clusterIDRaw := c.Query("cluster_id")

	var hosts []models.Host
	query := h.DB.Model(&models.Host{})
	query = h.filterByUser(c, query)


	if clusterIDRaw != "" {
		clusterID, err := strconv.ParseUint(clusterIDRaw, 10, 64)
		if err != nil {
			common.Error(c, http.StatusBadRequest, "cluster_id 参数无效")
			return
		}
		query = query.Where("cluster_id = ?", clusterID)
	}
	if err := query.Find(&hosts).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "查询主机失败: "+err.Error())
		return
	}

	hostIDs := make([]uint, 0, len(hosts))
	for _, host := range hosts {
		hostIDs = append(hostIDs, host.ID)
	}

	var apps []models.App
	if len(hostIDs) > 0 {
		// App 已经通过 hostID 被间接隔离了，但为了安全起见也可以显式隔离
		appQuery := h.DB.Model(&models.App{}).Where("host_id IN ?", hostIDs)
		appQuery = h.filterByUser(c, appQuery)
		if err := appQuery.Find(&apps).Error; err != nil {
			common.Error(c, http.StatusInternalServerError, "查询应用失败: "+err.Error())
			return
		}
	}

	appIDs := make([]uint, 0, len(apps))
	appIDSet := make(map[uint]struct{}, len(apps))
	for _, app := range apps {
		appIDs = append(appIDs, app.ID)
		appIDSet[app.ID] = struct{}{}
	}

	var deps []models.Dependency
	if len(appIDs) > 0 {
		depQuery := h.DB.Model(&models.Dependency{}).Where("source_app_id IN ? OR target_app_id IN ?", appIDs, appIDs)
		depQuery = h.filterByUser(c, depQuery)
		if err := depQuery.Find(&deps).Error; err != nil {
			common.Error(c, http.StatusInternalServerError, "查询依赖失败: "+err.Error())
			return
		}
	}

	nodes := make([]topologyNode, 0, len(hosts)+len(apps))
	links := make([]topologyLink, 0, len(apps)+len(deps))

	for _, host := range hosts {
		nodes = append(nodes, topologyNode{ID: "host-" + strconv.FormatUint(uint64(host.ID), 10), Name: host.Name, Type: "host"})
	}

	for _, app := range apps {
		nodeType := "app"
		switch app.Type {
		case "Database", "DB", "db":
			nodeType = "db"
		case "Cache", "cache":
			nodeType = "cache"
		}
		nodes = append(nodes, topologyNode{ID: "app-" + strconv.FormatUint(uint64(app.ID), 10), Name: app.Name, Type: nodeType})
		links = append(links, topologyLink{Source: "host-" + strconv.FormatUint(uint64(app.HostID), 10), Target: "app-" + strconv.FormatUint(uint64(app.ID), 10)})
	}

	for _, dep := range deps {
		if dep.SourceAppID != nil && dep.TargetAppID != nil {
			if _, ok := appIDSet[*dep.SourceAppID]; !ok {
				continue
			}
			if _, ok := appIDSet[*dep.TargetAppID]; !ok {
				continue
			}
			links = append(links, topologyLink{
				Source: "app-" + strconv.FormatUint(uint64(*dep.SourceAppID), 10),
				Target: "app-" + strconv.FormatUint(uint64(*dep.TargetAppID), 10),
			})
			continue
		}

		if dep.SourceHostID != nil && dep.TargetHostID != nil {
			links = append(links, topologyLink{
				Source: "host-" + strconv.FormatUint(uint64(*dep.SourceHostID), 10),
				Target: "host-" + strconv.FormatUint(uint64(*dep.TargetHostID), 10),
			})
		}
	}

	common.Success(c, gin.H{"nodes": nodes, "links": links})
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
