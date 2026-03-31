package handlers

import (
	"cmdb-v2/pkg/common"
	"cmdb-v2/pkg/models"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct{ DB *gorm.DB }

func New(db *gorm.DB) *Handler { return &Handler{DB: db} }

type exportPayload struct {
	Version      string              `json:"version"`
	ExportedAt   time.Time           `json:"exported_at"`
	Clusters     []models.Cluster    `json:"clusters"`
	Hosts        []models.Host       `json:"hosts"`
	Apps         []models.App        `json:"apps"`
	Ports        []models.Port       `json:"ports"`
	Domains      []models.Domain     `json:"domains"`
	Dependencies []models.Dependency `json:"dependencies"`
}

func (h *Handler) ExportCMDB(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uID, _ := userID.(uint)
	role, _ := c.Get("role")
	roleStr, _ := role.(string)

	payload, err := h.buildExportPayload(uID, roleStr)
	if err != nil {
		common.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	filename := "cmdb-export-" + time.Now().Format("20060102-150405") + ".json"
	record := buildTransferRecord(c, "export", "export", "success", "导出全部 JSON 成功", "导出了当前用户管理的 CMDB 全量数据")
	record.Filename = filename
	record.AddedSummary = marshalSummary(payload.summary())
	h.createTransferRecord(record)

	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.JSON(http.StatusOK, payload)
}

func (h *Handler) buildExportPayload(uID uint, role string) (exportPayload, error) {
	payload := exportPayload{
		Version:    "cmdb-export-v1",
		ExportedAt: time.Now(),
	}

	filter := func(q *gorm.DB) *gorm.DB {
		if role == "admin" {
			return q
		}
		return q.Where("user_id = ?", uID)
	}

	if err := filter(h.DB.Model(&models.Cluster{})).Order("id ASC").Find(&payload.Clusters).Error; err != nil {
		return payload, &importError{message: "导出集群失败: " + err.Error()}
	}
	if err := filter(h.DB.Model(&models.Host{})).Order("id ASC").Find(&payload.Hosts).Error; err != nil {
		return payload, &importError{message: "导出主机失败: " + err.Error()}
	}
	if err := filter(h.DB.Model(&models.App{})).Order("id ASC").Find(&payload.Apps).Error; err != nil {
		return payload, &importError{message: "导出应用失败: " + err.Error()}
	}
	if err := filter(h.DB.Model(&models.Port{})).Order("id ASC").Find(&payload.Ports).Error; err != nil {
		return payload, &importError{message: "导出端口失败: " + err.Error()}
	}
	if err := filter(h.DB.Model(&models.Domain{})).Order("id ASC").Find(&payload.Domains).Error; err != nil {
		return payload, &importError{message: "导出域名失败: " + err.Error()}
	}
	if err := filter(h.DB.Model(&models.Dependency{})).Order("id ASC").Find(&payload.Dependencies).Error; err != nil {
		return payload, &importError{message: "导出依赖失败: " + err.Error()}
	}

	return payload, nil
}

func (h *Handler) PreviewImportCMDB(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uID, _ := userID.(uint)
	role, _ := c.Get("role")
	roleStr, _ := role.(string)

	var payload exportPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		record := buildTransferRecord(c, "preview", "preview", "failed", "导入预览失败", "JSON 解析失败: "+err.Error())
		h.createTransferRecord(record)
		common.Error(c, http.StatusBadRequest, "导入文件格式错误: "+err.Error())
		return
	}
	if err := h.validateImportPayload(&payload); err != nil {
		record := buildTransferRecord(c, "preview", "preview", "failed", "导入预览失败", err.Error())
		record.IncomingSummary = marshalSummary(payload.summary())
		h.createTransferRecord(record)
		common.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.PreviewCMDB(uID, roleStr, &payload)
	if err != nil {
		record := buildTransferRecord(c, "preview", "preview", "failed", "导入预览失败", err.Error())
		record.IncomingSummary = marshalSummary(payload.summary())
		h.createTransferRecord(record)
		common.Error(c, http.StatusInternalServerError, "预览差异失败: "+err.Error())
		return
	}

	record := buildTransferRecord(c, "preview", "preview", "success", "导入预览完成", "已生成覆盖导入与追加导入差异预览")
	record.AddedSummary = marshalSummary(resp.Append.Summary)
	record.CurrentSummary = marshalSummary(resp.Overwrite.Current)
	record.IncomingSummary = marshalSummary(resp.Overwrite.Incoming)
	h.createTransferRecord(record)
	common.Success(c, resp)
}

func (h *Handler) ImportCMDB(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uID, _ := userID.(uint)
	role, _ := c.Get("role")
	roleStr, _ := role.(string)

	var payload exportPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		record := buildTransferRecord(c, "import", strings.ToLower(strings.TrimSpace(c.DefaultQuery("mode", "overwrite"))), "failed", "导入失败", "JSON 解析失败: "+err.Error())
		h.createTransferRecord(record)
		common.Error(c, http.StatusBadRequest, "导入文件格式错误: "+err.Error())
		return
	}
	if err := h.validateImportPayload(&payload); err != nil {
		record := buildTransferRecord(c, "import", strings.ToLower(strings.TrimSpace(c.DefaultQuery("mode", "overwrite"))), "failed", "导入失败", err.Error())
		record.IncomingSummary = marshalSummary(payload.summary())
		h.createTransferRecord(record)
		common.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	mode := strings.ToLower(strings.TrimSpace(c.DefaultQuery("mode", "overwrite")))
	if mode == "append" {
		result, err := h.AppendImportCMDB(uID, roleStr, &payload)
		if err != nil {
			record := buildTransferRecord(c, "import", mode, "failed", "追加导入失败", err.Error())
			record.IncomingSummary = marshalSummary(payload.summary())
			h.createTransferRecord(record)
			common.Error(c, http.StatusInternalServerError, "导入失败: "+err.Error())
			return
		}
		record := buildTransferRecord(c, "import", mode, "success", "追加导入完成", "保留现有数据，仅导入新增内容")
		record.AddedSummary = marshalSummary(result.Added)
		record.SkippedSummary = marshalSummary(result.Skipped)
		record.IncomingSummary = marshalSummary(payload.summary())
		h.createTransferRecord(record)
		common.Success(c, result)
		return
	}
	if mode != "overwrite" {
		record := buildTransferRecord(c, "import", mode, "failed", "导入失败", "不支持的导入模式: "+mode)
		record.IncomingSummary = marshalSummary(payload.summary())
		h.createTransferRecord(record)
		common.Error(c, http.StatusBadRequest, "不支持的导入模式: "+mode)
		return
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		// 覆盖导入仅针对当前用户的数据
		filter := func(q *gorm.DB) *gorm.DB {
			if roleStr == "admin" {
				return q
			}
			return q.Where("user_id = ?", uID)
		}

		if err := filter(tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped()).Delete(&models.Dependency{}).Error; err != nil {
			return err
		}
		if err := filter(tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped()).Delete(&models.Domain{}).Error; err != nil {
			return err
		}
		if err := filter(tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped()).Delete(&models.Port{}).Error; err != nil {
			return err
		}
		if err := filter(tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped()).Delete(&models.App{}).Error; err != nil {
			return err
		}
		if err := filter(tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped()).Delete(&models.Host{}).Error; err != nil {
			return err
		}
		if err := filter(tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped()).Delete(&models.Cluster{}).Error; err != nil {
			return err
		}

		// 强制为导入项设置 user_id
		for i := range payload.Clusters {
			payload.Clusters[i].UserID = uID
		}
		for i := range payload.Hosts {
			payload.Hosts[i].UserID = uID
		}
		for i := range payload.Apps {
			payload.Apps[i].UserID = uID
		}
		for i := range payload.Ports {
			payload.Ports[i].UserID = uID
		}
		for i := range payload.Domains {
			payload.Domains[i].UserID = uID
		}
		for i := range payload.Dependencies {
			payload.Dependencies[i].UserID = uID
		}

		insert := func(items interface{}) error {
			bytes, err := json.Marshal(items)
			if err != nil {
				return err
			}
			if string(bytes) == "null" || string(bytes) == "[]" {
				return nil
			}
			return tx.Create(items).Error
		}

		if err := insert(&payload.Clusters); err != nil {
			return err
		}
		if err := insert(&payload.Hosts); err != nil {
			return err
		}
		if err := insert(&payload.Apps); err != nil {
			return err
		}
		if err := insert(&payload.Ports); err != nil {
			return err
		}
		if err := insert(&payload.Domains); err != nil {
			return err
		}
		if err := insert(&payload.Dependencies); err != nil {
			return err
		}
		return nil
	}); err != nil {
		record := buildTransferRecord(c, "import", mode, "failed", "覆盖导入失败", err.Error())
		record.IncomingSummary = marshalSummary(payload.summary())
		h.createTransferRecord(record)
		common.Error(c, http.StatusInternalServerError, "导入失败: "+err.Error())
		return
	}

	result := importExecutionResult{
		Mode: "overwrite",
		Added: importSummary{
			Clusters:     len(payload.Clusters),
			Hosts:        len(payload.Hosts),
			Apps:         len(payload.Apps),
			Ports:        len(payload.Ports),
			Domains:      len(payload.Domains),
			Dependencies: len(payload.Dependencies),
		},
	}

	record := buildTransferRecord(c, "import", mode, "success", "覆盖导入完成", "已使用备份文件完整覆盖当前用户的 CMDB 数据")
	record.AddedSummary = marshalSummary(result.Added)
	record.IncomingSummary = marshalSummary(payload.summary())
	h.createTransferRecord(record)

	common.Success(c, result)
}

func (h *Handler) DownloadTemplate(c *gin.Context) {
	common.Success(c, gin.H{"message": "模板下载功能暂未实现"})
}

func (h *Handler) validateImportPayload(payload *exportPayload) error {
	if payload.Version == "" {
		return &importError{message: "导入文件缺少 version"}
	}
	if payload.Version != "cmdb-export-v1" {
		return &importError{message: "暂不支持该导入文件版本: " + payload.Version}
	}

	clusterIDs := make(map[uint]struct{}, len(payload.Clusters))
	hostIDs := make(map[uint]struct{}, len(payload.Hosts))
	appIDs := make(map[uint]struct{}, len(payload.Apps))

	for _, item := range payload.Clusters {
		clusterIDs[item.ID] = struct{}{}
	}
	for _, item := range payload.Hosts {
		if item.ClusterID != nil {
			if _, ok := clusterIDs[*item.ClusterID]; !ok {
				return &importError{message: "导入数据校验失败: 主机关联的集群不存在"}
			}
		}
		hostIDs[item.ID] = struct{}{}
	}
	for _, item := range payload.Apps {
		if _, ok := hostIDs[item.HostID]; !ok {
			return &importError{message: "导入数据校验失败: 应用关联的主机不存在"}
		}
		appIDs[item.ID] = struct{}{}
	}
	for _, item := range payload.Ports {
		if _, ok := appIDs[item.AppID]; !ok {
			return &importError{message: "导入数据校验失败: 端口关联的应用不存在"}
		}
	}
	for _, item := range payload.Domains {
		if item.AppID != nil {
			if _, ok := appIDs[*item.AppID]; !ok {
				return &importError{message: "导入数据校验失败: 域名关联的应用不存在"}
			}
		}
		if item.HostID != nil {
			if _, ok := hostIDs[*item.HostID]; !ok {
				return &importError{message: "导入数据校验失败: 域名关联的主机不存在"}
			}
		}
	}
	for _, item := range payload.Dependencies {
		if item.SourceAppID != nil {
			if _, ok := appIDs[*item.SourceAppID]; !ok {
				return &importError{message: "导入数据校验失败: 依赖调用方应用不存在"}
			}
		}
		if item.TargetAppID != nil {
			if _, ok := appIDs[*item.TargetAppID]; !ok {
				return &importError{message: "导入数据校验失败: 依赖被调用应用不存在"}
			}
		}
		if item.SourceHostID != nil {
			if _, ok := hostIDs[*item.SourceHostID]; !ok {
				return &importError{message: "导入数据校验失败: 依赖调用方主机不存在"}
			}
		}
		if item.TargetHostID != nil {
			if _, ok := hostIDs[*item.TargetHostID]; !ok {
				return &importError{message: "导入数据校验失败: 依赖被调用方主机不存在"}
			}
		}
	}

	return nil
}

type importError struct {
	message string
}

func (e *importError) Error() string {
	return e.message
}
