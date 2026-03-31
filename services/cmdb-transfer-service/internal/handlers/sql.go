package handlers

import (
	"cmdb-v2/pkg/common"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type sqlConsoleRequest struct {
	SQL string `json:"sql"`
}

type sqlQueryResponse struct {
	StatementType string                   `json:"statement_type"`
	Columns       []string                 `json:"columns"`
	Rows          []map[string]interface{} `json:"rows"`
	RowCount      int                      `json:"row_count"`
	AffectedRows  int64                    `json:"affected_rows"`
	ElapsedMS     int64                    `json:"elapsed_ms"`
	Truncated     bool                     `json:"truncated"`
}

const sqlPreviewLimit = 200

func (h *Handler) QuerySQL(c *gin.Context) {
	sqlText, statementType, err := parseSQLRequest(c)
	if err != nil {
		common.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if !isReadOnlyStatement(statementType) {
		common.Error(c, http.StatusBadRequest, "只读模式仅支持 SELECT / SHOW / DESC / EXPLAIN")
		return
	}

	start := time.Now()
	rows, columns, truncated, err := h.runQuery(sqlText)
	if err != nil {
		h.recordSQLAction(c, "query", "failed", err.Error(), sqlText)
		common.Error(c, http.StatusBadRequest, "SQL 查询失败: "+err.Error())
		return
	}

	resp := sqlQueryResponse{
		StatementType: statementType,
		Columns:       columns,
		Rows:          rows,
		RowCount:      len(rows),
		ElapsedMS:     time.Since(start).Milliseconds(),
		Truncated:     truncated,
	}
	h.recordSQLAction(c, "query", "success", fmt.Sprintf("返回 %d 行结果", len(rows)), sqlText)
	common.Success(c, resp)
}

func (h *Handler) ExecuteSQL(c *gin.Context) {
	sqlText, statementType, err := parseSQLRequest(c)
	if err != nil {
		common.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if isReadOnlyStatement(statementType) {
		common.Error(c, http.StatusBadRequest, "执行模式不支持只读语句，请使用查询模式")
		return
	}

	start := time.Now()
	result := h.DB.Exec(sqlText)
	if result.Error != nil {
		h.recordSQLAction(c, "execute", "failed", result.Error.Error(), sqlText)
		common.Error(c, http.StatusBadRequest, "SQL 执行失败: "+result.Error.Error())
		return
	}

	resp := sqlQueryResponse{
		StatementType: statementType,
		AffectedRows:  result.RowsAffected,
		ElapsedMS:     time.Since(start).Milliseconds(),
	}
	h.recordSQLAction(c, "execute", "success", fmt.Sprintf("影响 %d 行", result.RowsAffected), sqlText)
	common.Success(c, resp)
}

func parseSQLRequest(c *gin.Context) (string, string, error) {
	var req sqlConsoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return "", "", fmt.Errorf("请求参数错误")
	}

	sqlText := normalizeSQL(req.SQL)
	if sqlText == "" {
		return "", "", fmt.Errorf("SQL 不能为空")
	}
	if !isSingleStatement(sqlText) {
		return "", "", fmt.Errorf("仅支持执行单条 SQL 语句")
	}

	statementType := firstSQLKeyword(sqlText)
	if statementType == "" {
		return "", "", fmt.Errorf("无法识别 SQL 类型")
	}
	if isBlockedSQLStatement(statementType) {
		return "", "", fmt.Errorf("不支持执行该 SQL 类型")
	}

	return sqlText, statementType, nil
}

func normalizeSQL(sqlText string) string {
	sqlText = strings.TrimSpace(sqlText)
	sqlText = strings.TrimSuffix(sqlText, ";")
	return strings.TrimSpace(sqlText)
}

func isSingleStatement(sqlText string) bool {
	return !strings.Contains(sqlText, ";")
}

func firstSQLKeyword(sqlText string) string {
	parts := strings.Fields(strings.TrimSpace(sqlText))
	if len(parts) == 0 {
		return ""
	}
	return strings.ToUpper(parts[0])
}

func isReadOnlyStatement(statementType string) bool {
	switch statementType {
	case "SELECT", "SHOW", "DESC", "DESCRIBE", "EXPLAIN":
		return true
	default:
		return false
	}
}

func isBlockedSQLStatement(statementType string) bool {
	switch statementType {
	case "USE", "BEGIN", "START", "COMMIT", "ROLLBACK", "GRANT", "REVOKE":
		return true
	default:
		return false
	}
}

func (h *Handler) runQuery(sqlText string) ([]map[string]interface{}, []string, bool, error) {
	rows, err := h.DB.Raw(sqlText).Rows()
	if err != nil {
		return nil, nil, false, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, false, err
	}

	result := make([]map[string]interface{}, 0)
	truncated := false
	for rows.Next() {
		values := make([]interface{}, len(columns))
		scanTargets := make([]interface{}, len(columns))
		for i := range values {
			scanTargets[i] = &values[i]
		}

		if err := rows.Scan(scanTargets...); err != nil {
			return nil, nil, false, err
		}

		row := make(map[string]interface{}, len(columns))
		for idx, column := range columns {
			row[column] = normalizeSQLValue(values[idx])
		}
		result = append(result, row)

		if len(result) >= sqlPreviewLimit {
			truncated = rows.Next()
			break
		}
	}

	return result, columns, truncated, nil
}

func normalizeSQLValue(value interface{}) interface{} {
	switch item := value.(type) {
	case []byte:
		return string(item)
	case time.Time:
		return item.Format(time.RFC3339)
	default:
		return item
	}
}

func (h *Handler) recordSQLAction(c *gin.Context, mode string, status string, message string, sqlText string) {
	record := buildTransferRecord(c, "sql", mode, status, message, sqlText)
	record.Filename = ""
	h.createTransferRecord(record)
}
