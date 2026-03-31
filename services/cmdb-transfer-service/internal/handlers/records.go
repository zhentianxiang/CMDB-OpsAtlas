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

type transferRecordResponse struct {
	ID        uint          `json:"id"`
	Action    string        `json:"action"`
	Mode      string        `json:"mode"`
	Status    string        `json:"status"`
	Filename  string        `json:"filename"`
	Operator  string        `json:"operator"`
	Message   string        `json:"message"`
	Detail    string        `json:"detail"`
	Added     importSummary `json:"added"`
	Skipped   importSummary `json:"skipped"`
	Current   importSummary `json:"current"`
	Incoming  importSummary `json:"incoming"`
	CreatedAt time.Time     `json:"created_at"`
}

func (h *Handler) ListTransferRecords(c *gin.Context) {
	var records []models.TransferRecord
	query := h.DB.Model(&models.TransferRecord{}).Order("created_at DESC")

	// 数据隔离
	query = h.filterByUser(c, query)


	action := strings.TrimSpace(c.Query("action"))
	if action != "" {
		query = query.Where("action = ?", action)
	}

	limit := 50
	if err := query.Limit(limit).Find(&records).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "查询操作记录失败: "+err.Error())
		return
	}

	result := make([]transferRecordResponse, 0, len(records))
	for _, item := range records {
		result = append(result, transferRecordResponse{
			ID:        item.ID,
			Action:    item.Action,
			Mode:      item.Mode,
			Status:    item.Status,
			Filename:  item.Filename,
			Operator:  item.Operator,
			Message:   item.Message,
			Detail:    item.Detail,
			Added:     parseSummary(item.AddedSummary),
			Skipped:   parseSummary(item.SkippedSummary),
			Current:   parseSummary(item.CurrentSummary),
			Incoming:  parseSummary(item.IncomingSummary),
			CreatedAt: item.CreatedAt,
		})
	}

	common.Success(c, result)
}

func (h *Handler) GetTransferRecord(c *gin.Context) {
	common.Success(c, gin.H{"message": "获取记录详情功能暂未实现"})
}

func (h *Handler) createTransferRecord(record models.TransferRecord) {
	if err := h.DB.Create(&record).Error; err != nil {
		return
	}
}

func buildTransferRecord(c *gin.Context, action string, mode string, status string, message string, detail string) models.TransferRecord {
	operator := strings.TrimSpace(c.GetString("username"))
	if operator == "" {
		operator = "unknown"
	}

	userID, _ := c.Get("user_id")
	uID, _ := userID.(uint)

	filename := strings.TrimSpace(c.GetHeader("X-CMDB-Filename"))

	return models.TransferRecord{
		UserID:   uID,
		Action:   action,
		Mode:     mode,
		Status:   status,
		Filename: filename,
		Operator: operator,
		Message:  message,
		Detail:   detail,
	}
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

func marshalSummary(summary importSummary) string {
	if summary == (importSummary{}) {
		return ""
	}
	bytes, err := json.Marshal(summary)
	if err != nil {
		return ""
	}
	return string(bytes)
}

func parseSummary(value string) importSummary {
	if strings.TrimSpace(value) == "" {
		return importSummary{}
	}
	var summary importSummary
	if err := json.Unmarshal([]byte(value), &summary); err != nil {
		return importSummary{}
	}
	return summary
}
