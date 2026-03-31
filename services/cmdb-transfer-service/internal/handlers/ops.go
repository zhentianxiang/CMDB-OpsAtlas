package handlers

import (
	"cmdb-v2/pkg/common"
	"cmdb-v2/pkg/models"
	"fmt"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
)

type opsCountItem struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type opsServiceStatus struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	LatencyMS int64  `json:"latency_ms"`
	URL       string `json:"url"`
	Message   string `json:"message"`
}

type opsResourceTotals struct {
	TotalCPU     int `json:"total_cpu"`
	TotalMemory  int `json:"total_memory"`
	OnlineHosts  int `json:"online_hosts"`
	OfflineHosts int `json:"offline_hosts"`
}

type opsLatestApp struct {
	ID         uint      `json:"id"`
	Name       string    `json:"name"`
	HostName   string    `json:"host_name"`
	Type       string    `json:"type"`
	DeployType string    `json:"deploy_type"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type opsLatestDomain struct {
	ID        uint      `json:"id"`
	Domain    string    `json:"domain"`
	AppName   string    `json:"app_name"`
	HostName  string    `json:"host_name"`
	UpdatedAt time.Time `json:"updated_at"`
}

type opsLatestDependency struct {
	ID        uint      `json:"id"`
	Source    string    `json:"source"`
	Target    string    `json:"target"`
	Desc      string    `json:"desc"`
	UpdatedAt time.Time `json:"updated_at"`
}

type opsOverviewResponse struct {
	Counts         importSummary         `json:"counts"`
	HostStatus     []opsCountItem        `json:"host_status"`
	AppTypes       []opsCountItem        `json:"app_types"`
	DeployTypes    []opsCountItem        `json:"deploy_types"`
	ResourceTotals opsResourceTotals     `json:"resource_totals"`
	Services       []opsServiceStatus    `json:"services"`
	LatestApps     []opsLatestApp        `json:"latest_apps"`
	LatestDomains  []opsLatestDomain     `json:"latest_domains"`
	LatestDeps     []opsLatestDependency `json:"latest_dependencies"`
}

func (h *Handler) GetOpsOverview(c *gin.Context) {
	var (
		clusters     []models.Cluster
		hosts        []models.Host
		apps         []models.App
		ports        []models.Port
		domains      []models.Domain
		dependencies []models.Dependency
	)

	if err := h.DB.Find(&clusters).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "查询集群失败: "+err.Error())
		return
	}
	if err := h.DB.Find(&hosts).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "查询主机失败: "+err.Error())
		return
	}
	if err := h.DB.Find(&apps).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "查询应用失败: "+err.Error())
		return
	}
	if err := h.DB.Find(&ports).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "查询端口失败: "+err.Error())
		return
	}
	if err := h.DB.Find(&domains).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "查询域名失败: "+err.Error())
		return
	}
	if err := h.DB.Find(&dependencies).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "查询依赖失败: "+err.Error())
		return
	}

	hostNameMap := make(map[uint]string, len(hosts))
	resourceTotals := opsResourceTotals{}
	hostStatusMap := map[string]int{}
	for _, item := range hosts {
		hostNameMap[item.ID] = item.Name
		resourceTotals.TotalCPU += item.CPU
		resourceTotals.TotalMemory += item.Memory
		status := item.Status
		if status == "" {
			status = "unknown"
		}
		hostStatusMap[status]++
		if status == "online" || status == "running" || status == "healthy" {
			resourceTotals.OnlineHosts++
		} else {
			resourceTotals.OfflineHosts++
		}
	}

	appNameMap := make(map[uint]string, len(apps))
	appTypeMap := map[string]int{}
	deployTypeMap := map[string]int{}
	for _, item := range apps {
		appNameMap[item.ID] = item.Name
		appType := item.Type
		if appType == "" {
			appType = "未分类"
		}
		deployType := item.DeployType
		if deployType == "" {
			deployType = "未设置"
		}
		appTypeMap[appType]++
		deployTypeMap[deployType]++
	}

	latestApps := make([]models.App, len(apps))
	copy(latestApps, apps)
	sort.Slice(latestApps, func(i, j int) bool {
		return latestApps[i].UpdatedAt.After(latestApps[j].UpdatedAt)
	})
	if len(latestApps) > 8 {
		latestApps = latestApps[:8]
	}

	latestDomainsRaw := make([]models.Domain, len(domains))
	copy(latestDomainsRaw, domains)
	sort.Slice(latestDomainsRaw, func(i, j int) bool {
		return latestDomainsRaw[i].UpdatedAt.After(latestDomainsRaw[j].UpdatedAt)
	})
	if len(latestDomainsRaw) > 8 {
		latestDomainsRaw = latestDomainsRaw[:8]
	}

	latestDepsRaw := make([]models.Dependency, len(dependencies))
	copy(latestDepsRaw, dependencies)
	sort.Slice(latestDepsRaw, func(i, j int) bool {
		return latestDepsRaw[i].UpdatedAt.After(latestDepsRaw[j].UpdatedAt)
	})
	if len(latestDepsRaw) > 8 {
		latestDepsRaw = latestDepsRaw[:8]
	}

	resp := opsOverviewResponse{
		Counts: importSummary{
			Clusters:     len(clusters),
			Hosts:        len(hosts),
			Apps:         len(apps),
			Ports:        len(ports),
			Domains:      len(domains),
			Dependencies: len(dependencies),
		},
		HostStatus:     mapToCountItems(hostStatusMap),
		AppTypes:       mapToCountItems(appTypeMap),
		DeployTypes:    mapToCountItems(deployTypeMap),
		ResourceTotals: resourceTotals,
		Services:       collectServiceStatuses(),
	}

	for _, item := range latestApps {
		resp.LatestApps = append(resp.LatestApps, opsLatestApp{
			ID:         item.ID,
			Name:       item.Name,
			HostName:   hostNameMap[item.HostID],
			Type:       item.Type,
			DeployType: item.DeployType,
			UpdatedAt:  item.UpdatedAt,
		})
	}

	for _, item := range latestDomainsRaw {
		appName := ""
		hostName := ""
		if item.AppID != nil {
			appName = appNameMap[*item.AppID]
		}
		if item.HostID != nil {
			hostName = hostNameMap[*item.HostID]
		}
		resp.LatestDomains = append(resp.LatestDomains, opsLatestDomain{
			ID:        item.ID,
			Domain:    item.Domain,
			AppName:   appName,
			HostName:  hostName,
			UpdatedAt: item.UpdatedAt,
		})
	}

	for _, item := range latestDepsRaw {
		resp.LatestDeps = append(resp.LatestDeps, opsLatestDependency{
			ID:        item.ID,
			Source:    dependencyEndpointText(item.SourceAppID, item.SourceHostID, item.SourceNode, appNameMap, hostNameMap),
			Target:    dependencyEndpointText(item.TargetAppID, item.TargetHostID, item.TargetNode, appNameMap, hostNameMap),
			Desc:      item.Desc,
			UpdatedAt: item.UpdatedAt,
		})
	}

	common.Success(c, resp)
}

func mapToCountItems(source map[string]int) []opsCountItem {
	items := make([]opsCountItem, 0, len(source))
	for key, value := range source {
		items = append(items, opsCountItem{Name: key, Value: value})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Value == items[j].Value {
			return items[i].Name < items[j].Name
		}
		return items[i].Value > items[j].Value
	})
	return items
}

func dependencyEndpointText(appID *uint, hostID *uint, node string, appNameMap map[uint]string, hostNameMap map[uint]string) string {
	if appID != nil {
		if name := appNameMap[*appID]; name != "" {
			return "应用: " + name
		}
		return fmt.Sprintf("应用ID: %d", *appID)
	}
	if hostID != nil {
		if name := hostNameMap[*hostID]; name != "" {
			return "主机: " + name
		}
		return fmt.Sprintf("主机ID: %d", *hostID)
	}
	if node != "" {
		return "外部: " + node
	}
	return "-"
}

func collectServiceStatuses() []opsServiceStatus {
	services := []struct {
		name string
		url  string
	}{
		{"auth-service", envOrDefault("AUTH_SERVICE_HEALTH_URL", "http://auth-service:8080/healthz")},
		{"cluster-service", envOrDefault("CLUSTER_SERVICE_HEALTH_URL", "http://cluster-service:8080/healthz")},
		{"host-service", envOrDefault("HOST_SERVICE_HEALTH_URL", "http://host-service:8080/healthz")},
		{"app-service", envOrDefault("APP_SERVICE_HEALTH_URL", "http://app-service:8080/healthz")},
		{"port-service", envOrDefault("PORT_SERVICE_HEALTH_URL", "http://port-service:8080/healthz")},
		{"domain-service", envOrDefault("DOMAIN_SERVICE_HEALTH_URL", "http://domain-service:8080/healthz")},
		{"dependency-service", envOrDefault("DEPENDENCY_SERVICE_HEALTH_URL", "http://dependency-service:8080/healthz")},
		{"topology-service", envOrDefault("TOPOLOGY_SERVICE_HEALTH_URL", "http://topology-service:8080/healthz")},
		{"cmdb-transfer-service", envOrDefault("CMDB_TRANSFER_SERVICE_HEALTH_URL", "http://cmdb-transfer-service:8080/healthz")},
	}

	client := &http.Client{Timeout: 2 * time.Second}
	results := make([]opsServiceStatus, 0, len(services))
	for _, item := range services {
		begin := time.Now()
		resp, err := client.Get(item.url)
		latency := time.Since(begin).Milliseconds()
		if err != nil {
			results = append(results, opsServiceStatus{
				Name:      item.name,
				Status:    "DOWN",
				LatencyMS: latency,
				URL:       item.url,
				Message:   err.Error(),
			})
			continue
		}
		_ = resp.Body.Close()
		status := "UP"
		message := resp.Status
		if resp.StatusCode >= 400 {
			status = "DEGRADED"
		}
		results = append(results, opsServiceStatus{
			Name:      item.name,
			Status:    status,
			LatencyMS: latency,
			URL:       item.url,
			Message:   message,
		})
	}
	return results
}

func envOrDefault(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
