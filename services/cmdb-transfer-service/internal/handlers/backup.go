package handlers

import (
	"cmdb-v2/pkg/common"
	"cmdb-v2/pkg/models"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

type backupPolicyPayload struct {
	Enabled       bool     `json:"enabled"`
	BackupHour    int      `json:"backup_hour"`
	RetentionDays int      `json:"retention_days"`
	BackupTypes   []string `json:"backup_types"`
	BackupDir     string   `json:"backup_dir"`
}

type backupPolicyResponse struct {
	ID            uint       `json:"id"`
	Enabled       bool       `json:"enabled"`
	BackupHour    int        `json:"backup_hour"`
	RetentionDays int        `json:"retention_days"`
	BackupTypes   []string   `json:"backup_types"`
	BackupDir     string     `json:"backup_dir"`
	LastRunAt     *time.Time `json:"last_run_at"`
}

type backupFileResponse struct {
	ID            uint       `json:"id"`
	BatchNo       string     `json:"batch_no"`
	TriggerSource string     `json:"trigger_source"`
	BackupType    string     `json:"backup_type"`
	Status        string     `json:"status"`
	Filename      string     `json:"filename"`
	SizeBytes     int64      `json:"size_bytes"`
	Message       string     `json:"message"`
	Operator      string     `json:"operator"`
	StartedAt     time.Time  `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at"`
	ExpiresAt     *time.Time `json:"expires_at"`
}

type manualBackupRequest struct {
	TriggerSource string `json:"trigger_source"`
}

func (h *Handler) StartBackupScheduler() {
	go func() {
		h.runScheduledBackup()
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			h.runScheduledBackup()
		}
	}()
}

func (h *Handler) GetBackupPolicy(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uID, _ := userID.(uint)

	policy, err := h.getOrCreateBackupPolicy(uID)
	if err != nil {
		common.Error(c, http.StatusInternalServerError, "查询备份策略失败: "+err.Error())
		return
	}
	common.Success(c, toBackupPolicyResponse(policy))
}

func (h *Handler) UpdateBackupPolicy(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uID, _ := userID.(uint)

	var payload backupPolicyPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		common.Error(c, http.StatusBadRequest, "备份策略参数错误: "+err.Error())
		return
	}
	if err := validateBackupPolicyPayload(payload); err != nil {
		common.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	policy, err := h.getOrCreateBackupPolicy(uID)
	if err != nil {
		common.Error(c, http.StatusInternalServerError, "查询备份策略失败: "+err.Error())
		return
	}

	policy.Enabled = payload.Enabled
	policy.BackupHour = payload.BackupHour
	policy.RetentionDays = payload.RetentionDays
	policy.BackupTypes = strings.Join(normalizeBackupTypes(payload.BackupTypes), ",")
	policy.BackupDir = strings.TrimSpace(payload.BackupDir)
	if err := os.MkdirAll(policy.BackupDir, 0o755); err != nil {
		common.Error(c, http.StatusInternalServerError, "创建备份目录失败: "+err.Error())
		return
	}
	if err := h.DB.Save(policy).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "保存备份策略失败: "+err.Error())
		return
	}

	common.Success(c, toBackupPolicyResponse(policy))
}

func (h *Handler) ListBackupFiles(c *gin.Context) {
	var files []models.BackupFile
	query := h.DB.Model(&models.BackupFile{}).Order("started_at DESC")

	// 数据隔离
	query = h.filterByUser(c, query)


	if backupType := strings.TrimSpace(c.Query("type")); backupType != "" {
		query = query.Where("backup_type = ?", backupType)
	}
	if err := query.Limit(100).Find(&files).Error; err != nil {
		common.Error(c, http.StatusInternalServerError, "查询备份文件失败: "+err.Error())
		return
	}

	result := make([]backupFileResponse, 0, len(files))
	for _, item := range files {
		result = append(result, backupFileResponse{
			ID:            item.ID,
			BatchNo:       item.BatchNo,
			TriggerSource: item.TriggerSource,
			BackupType:    item.BackupType,
			Status:        item.Status,
			Filename:      item.Filename,
			SizeBytes:     item.SizeBytes,
			Message:       item.Message,
			Operator:      item.Operator,
			StartedAt:     item.StartedAt,
			CompletedAt:   item.CompletedAt,
			ExpiresAt:     item.ExpiresAt,
		})
	}

	common.Success(c, result)
}

func (h *Handler) RunBackupNow(c *gin.Context) {
	var req manualBackupRequest
	_ = c.ShouldBindJSON(&req)

	userID, _ := c.Get("user_id")
	uID, _ := userID.(uint)

	operator := strings.TrimSpace(c.GetString("username"))
	if operator == "" {
		operator = "manual"
	}
	source := strings.TrimSpace(req.TriggerSource)
	if source == "" {
		source = "manual"
	}

	result, err := h.runBackupTask(uID, source, operator)
	if err != nil {
		common.Error(c, http.StatusInternalServerError, "执行备份失败: "+err.Error())
		return
	}
	common.Success(c, result)
}

func (h *Handler) DeleteBackupFile(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var file models.BackupFile
	if err := h.DB.First(&file, id).Error; err != nil {
		common.Error(c, http.StatusNotFound, "备份文件不存在")
		return
	}
	if file.FilePath != "" {
		_ = os.Remove(file.FilePath)
	}
	h.DB.Delete(&file)
	common.Success(c, nil)
}

func (h *Handler) DownloadBackupFile(c *gin.Context) {
	fileID, err := parseBackupFileID(c.Param("id"))
	if err != nil {
		common.Error(c, http.StatusBadRequest, "备份文件编号错误")
		return
	}
	var file models.BackupFile
	query := h.DB.Model(&models.BackupFile{})
	query = h.filterByUser(c, query)


	if err := query.First(&file, fileID).Error; err != nil {
		common.Error(c, http.StatusNotFound, "备份文件不存在")
		return
	}
	if file.Status != "success" {
		common.Error(c, http.StatusBadRequest, "该备份记录没有可下载文件")
		return
	}
	if _, err := os.Stat(file.FilePath); err != nil {
		common.Error(c, http.StatusNotFound, "备份文件不存在或已被清理")
		return
	}
	c.FileAttachment(file.FilePath, file.Filename)
}

func (h *Handler) RestoreBackupFile(c *gin.Context) {
	fileID, err := parseBackupFileID(c.Param("id"))
	if err != nil {
		common.Error(c, http.StatusBadRequest, "备份文件编号错误")
		return
	}

	var file models.BackupFile
	query := h.DB.Model(&models.BackupFile{})
	query = h.filterByUser(c, query)


	if err := query.First(&file, fileID).Error; err != nil {
		common.Error(c, http.StatusNotFound, "备份文件不存在")
		return
	}
	if file.BackupType != "database" {
		common.Error(c, http.StatusBadRequest, "当前仅支持 SQL 数据库备份一键恢复")
		return
	}
	if file.Status != "success" {
		common.Error(c, http.StatusBadRequest, "仅成功的 SQL 备份文件支持恢复")
		return
	}
	if _, err := os.Stat(file.FilePath); err != nil {
		common.Error(c, http.StatusNotFound, "备份文件不存在或已被清理")
		return
	}

	operator := strings.TrimSpace(c.GetString("username"))
	if operator == "" {
		operator = "unknown"
	}

	userID, _ := c.Get("user_id")
	uID, _ := userID.(uint)

	record := buildTransferRecordFromOperator(uID, operator, "restore", "database", "success", "SQL 备份恢复完成", "已使用数据库 SQL 备份文件恢复当前库")
	record.Filename = file.Filename

	if err := h.restoreDatabaseBackup(file.FilePath); err != nil {
		record.Status = "failed"
		record.Message = "SQL 备份恢复失败"
		record.Detail = err.Error()
		h.createTransferRecord(record)
		common.Error(c, http.StatusInternalServerError, "恢复 SQL 备份失败: "+err.Error())
		return
	}

	summary := importSummary{}
	payload, payloadErr := h.buildExportPayload(uID, "admin") // 管理员角色恢复时可能需要全局视角，但这里恢复给当前用户
	if payloadErr == nil {
		summary = payload.summary()
	}
	record.AddedSummary = marshalSummary(summary)
	h.createTransferRecord(record)

	common.Success(c, gin.H{
		"file_id":     file.ID,
		"filename":    file.Filename,
		"restored":    true,
		"message":     "SQL 备份恢复成功",
		"restored_at": time.Now(),
	})
}

func (h *Handler) runScheduledBackup() {
	// 定时任务需要遍历所有启用的策略
	var policies []models.BackupPolicy
	if err := h.DB.Where("enabled = ?", true).Find(&policies).Error; err != nil {
		return
	}

	for _, policy := range policies {
		now := time.Now()
		if now.Hour() != policy.BackupHour || now.Minute() != 0 {
			continue
		}
		if policy.LastRunAt != nil {
			last := policy.LastRunAt.In(now.Location())
			if last.Year() == now.Year() && last.YearDay() == now.YearDay() && last.Hour() == policy.BackupHour {
				continue
			}
		}

		if _, err := h.runBackupTask(policy.UserID, "scheduled", "system"); err != nil {
			continue
		}
	}
}

func (h *Handler) runBackupTask(uID uint, source string, operator string) ([]backupFileResponse, error) {
	policy, err := h.getOrCreateBackupPolicy(uID)
	if err != nil {
		return nil, err
	}

	backupTypes := normalizeBackupTypes(strings.Split(policy.BackupTypes, ","))
	backupDir := strings.TrimSpace(policy.BackupDir)
	if backupDir == "" {
		backupDir = defaultBackupDir()
	}
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return nil, err
	}

	batchNo := time.Now().Format("20060102-150405")
	expireAt := time.Now().Add(time.Duration(policy.RetentionDays) * 24 * time.Hour)
	results := make([]backupFileResponse, 0, len(backupTypes))
	failedTypes := make([]string, 0)

	for _, backupType := range backupTypes {
		record := models.BackupFile{
			UserID:        uID,
			BatchNo:       batchNo,
			TriggerSource: source,
			BackupType:    backupType,
			Status:        "running",
			Operator:      operator,
			StartedAt:     time.Now(),
			ExpiresAt:     &expireAt,
		}
		if err := h.DB.Create(&record).Error; err != nil {
			return nil, err
		}

		var runErr error
		switch backupType {
		case "json":
			runErr = h.createJSONBackup(uID, &record, backupDir, batchNo)
		case "database":
			runErr = h.createDatabaseBackup(&record, backupDir, batchNo)
		default:
			runErr = fmt.Errorf("不支持的备份类型: %s", backupType)
		}

		completedAt := time.Now()
		record.CompletedAt = &completedAt
		if runErr != nil {
			record.Status = "failed"
			record.Message = runErr.Error()
		} else {
			record.Status = "success"
		}
		if err := h.DB.Save(&record).Error; err != nil {
			return nil, err
		}
		if runErr != nil {
			failedTypes = append(failedTypes, backupType)
		}

		results = append(results, backupFileResponse{
			ID:            record.ID,
			BatchNo:       record.BatchNo,
			TriggerSource: record.TriggerSource,
			BackupType:    record.BackupType,
			Status:        record.Status,
			Filename:      record.Filename,
			SizeBytes:     record.SizeBytes,
			Message:       record.Message,
			Operator:      record.Operator,
			StartedAt:     record.StartedAt,
			CompletedAt:   record.CompletedAt,
			ExpiresAt:     record.ExpiresAt,
		})
	}

	policy.LastRunAt = timePtr(time.Now())
	if err := h.DB.Save(policy).Error; err != nil {
		return nil, err
	}

	if err := h.cleanupExpiredBackups(uID, policy.RetentionDays); err != nil {
		// keep backup success even if cleanup failed
	}

	summary := importSummary{}
	payload, payloadErr := h.buildExportPayload(uID, "admin") // 管理员身份导出用于备份
	if payloadErr == nil {
		summary = payload.summary()
	}
	status := "success"
	message := "执行备份完成"
	detail := fmt.Sprintf("已生成 %d 个备份文件", len(results))
	var resultErr error
	if len(failedTypes) > 0 {
		status = "failed"
		message = "执行备份失败"
		detail = "以下备份类型执行失败: " + strings.Join(failedTypes, ", ")
		resultErr = fmt.Errorf(detail)
	}
	record := buildTransferRecordFromOperator(uID, operator, "backup", source, status, message, detail)
	record.AddedSummary = marshalSummary(summary)
	h.createTransferRecord(record)

	return results, resultErr
}

func (h *Handler) createJSONBackup(uID uint, record *models.BackupFile, backupDir string, batchNo string) error {
	payload, err := h.buildExportPayload(uID, "admin") // 系统备份使用管理员视角
	if err != nil {
		return err
	}
	bytes, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	filename := fmt.Sprintf("cmdb-backup-%s-u%d.json", batchNo, uID)
	filepath := filepath.Join(backupDir, filename)
	if err := os.WriteFile(filepath, bytes, 0o644); err != nil {
		return err
	}
	info, err := os.Stat(filepath)
	if err != nil {
		return err
	}
	record.Filename = filename
	record.FilePath = filepath
	record.SizeBytes = info.Size()
	record.Message = "CMDB JSON 逻辑备份成功"
	return nil
}

func (h *Handler) createDatabaseBackup(record *models.BackupFile, backupDir string, batchNo string) error {
	// 数据库物理备份目前是全局的，为了简单我们暂时保留
	cfg, err := mysqlDriver.ParseDSN(getDBDSN())
	if err != nil {
		return err
	}
	host, port, err := parseAddr(cfg.Addr)
	if err != nil {
		return err
	}
	dumpBinary, err := findDumpBinary()
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("cmdb-backup-%s-sql.sql", batchNo)
	targetPath := filepath.Join(backupDir, filename)
	file, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer file.Close()

	cmd := exec.Command(dumpBinary, "-h", host, "-P", port, "-u", cfg.User, "--single-transaction", "--quick", cfg.DBName)
	cmd.Env = append(os.Environ(), "MYSQL_PWD="+cfg.Passwd)
	cmd.Stdout = file
	cmd.Stderr = file
	if err := cmd.Run(); err != nil {
		return err
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		return err
	}
	record.Filename = filename
	record.FilePath = targetPath
	record.SizeBytes = info.Size()
	record.Message = "MySQL SQL 物理备份成功"
	return nil
}

func (h *Handler) restoreDatabaseBackup(filePath string) error {
	cfg, err := mysqlDriver.ParseDSN(getDBDSN())
	if err != nil {
		return err
	}
	host, port, err := parseAddr(cfg.Addr)
	if err != nil {
		return err
	}
	mysqlBinary, err := findMySQLBinary()
	if err != nil {
		return err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	cmd := exec.Command(mysqlBinary, "-h", host, "-P", port, "-u", cfg.User, cfg.DBName)
	cmd.Env = append(os.Environ(), "MYSQL_PWD="+cfg.Passwd)
	cmd.Stdin = file
	output, err := cmd.CombinedOutput()
	if err != nil {
		text := strings.TrimSpace(string(output))
		if text == "" {
			return err
		}
		return fmt.Errorf("%v: %s", err, text)
	}

	return nil
}

func (h *Handler) cleanupExpiredBackups(uID uint, retentionDays int) error {
	if retentionDays <= 0 {
		return nil
	}
	cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour)
	var files []models.BackupFile
	if err := h.DB.Where("user_id = ? AND created_at < ?", uID, cutoff).Find(&files).Error; err != nil {
		return err
	}

	for _, item := range files {
		if item.FilePath != "" {
			_ = os.Remove(item.FilePath)
		}
	}
	return h.DB.Where("user_id = ? AND created_at < ?", uID, cutoff).Delete(&models.BackupFile{}).Error
}

func (h *Handler) getOrCreateBackupPolicy(uID uint) (*models.BackupPolicy, error) {
	var policy models.BackupPolicy
	err := h.DB.Where("user_id = ?", uID).Order("id ASC").First(&policy).Error
	if err == nil {
		changed := false
		if strings.TrimSpace(policy.BackupDir) == "" {
			policy.BackupDir = defaultBackupDir()
			changed = true
		}
		if strings.TrimSpace(policy.BackupTypes) == "" {
			policy.BackupTypes = "json,database"
			changed = true
		}
		if policy.BackupHour < 0 || policy.BackupHour > 23 {
			policy.BackupHour = 2
			changed = true
		}
		if policy.RetentionDays <= 0 {
			policy.RetentionDays = 7
			changed = true
		}
		if changed {
			if saveErr := h.DB.Save(&policy).Error; saveErr != nil {
				return nil, saveErr
			}
		}
		return &policy, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	policy = models.BackupPolicy{
		UserID:        uID,
		Enabled:       false,
		BackupHour:    2,
		RetentionDays: 7,
		BackupTypes:   "json,database",
		BackupDir:     defaultBackupDir(),
	}
	if err := os.MkdirAll(policy.BackupDir, 0o755); err != nil {
		return nil, err
	}
	if err := h.DB.Create(&policy).Error; err != nil {
		return nil, err
	}
	return &policy, nil
}

func validateBackupPolicyPayload(payload backupPolicyPayload) error {
	if payload.BackupHour < 0 || payload.BackupHour > 23 {
		return &importError{message: "备份执行时间必须在 0 到 23 点之间"}
	}
	if payload.RetentionDays < 1 || payload.RetentionDays > 365 {
		return &importError{message: "备份保留天数必须在 1 到 365 天之间"}
	}
	if strings.TrimSpace(payload.BackupDir) == "" {
		return &importError{message: "备份目录不能为空"}
	}
	types := normalizeBackupTypes(payload.BackupTypes)
	if len(types) == 0 {
		return &importError{message: "至少选择一种备份类型"}
	}
	return nil
}

func normalizeBackupTypes(values []string) []string {
	allowed := map[string]struct{}{
		"json":     {},
		"database": {},
	}
	items := make([]string, 0, len(values))
	for _, item := range values {
		normalized := strings.ToLower(strings.TrimSpace(item))
		if _, ok := allowed[normalized]; !ok {
			continue
		}
		if !slices.Contains(items, normalized) {
			items = append(items, normalized)
		}
	}
	if len(items) == 0 {
		return []string{"json", "database"}
	}
	return items
}

func defaultBackupDir() string {
	value := strings.TrimSpace(os.Getenv("BACKUP_DIR"))
	if value == "" {
		return "/app/backups"
	}
	return value
}

func toBackupPolicyResponse(policy *models.BackupPolicy) backupPolicyResponse {
	return backupPolicyResponse{
		ID:            policy.ID,
		Enabled:       policy.Enabled,
		BackupHour:    policy.BackupHour,
		RetentionDays: policy.RetentionDays,
		BackupTypes:   normalizeBackupTypes(strings.Split(policy.BackupTypes, ",")),
		BackupDir:     policy.BackupDir,
		LastRunAt:     policy.LastRunAt,
	}
}

func getDBDSN() string {
	dsn := strings.TrimSpace(os.Getenv("DB_DSN"))
	if dsn == "" {
		return "root:rootpassword@tcp(mysql:3306)/cmdb_resource?charset=utf8mb4&parseTime=True&loc=Local"
	}
	return dsn
}

func parseAddr(addr string) (string, string, error) {
	trimmed := strings.TrimPrefix(addr, "tcp(")
	trimmed = strings.TrimSuffix(trimmed, ")")
	host, port, err := net.SplitHostPort(trimmed)
	if err == nil {
		return host, port, nil
	}
	if strings.Contains(trimmed, ":") {
		parts := strings.Split(trimmed, ":")
		return parts[0], parts[1], nil
	}
	return trimmed, "3306", nil
}

func findDumpBinary() (string, error) {
	if path, err := exec.LookPath("mysqldump"); err == nil {
		return path, nil
	}
	if path, err := exec.LookPath("mariadb-dump"); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("当前环境未安装 mysqldump")
}

func findMySQLBinary() (string, error) {
	if path, err := exec.LookPath("mysql"); err == nil {
		return path, nil
	}
	if path, err := exec.LookPath("mariadb"); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("当前环境未安装 mysql 客户端")
}

func timePtr(value time.Time) *time.Time {
	return &value
}

func buildTransferRecordFromOperator(uID uint, operator string, action string, mode string, status string, message string, detail string) models.TransferRecord {
	if strings.TrimSpace(operator) == "" {
		operator = "unknown"
	}
	return models.TransferRecord{
		UserID:   uID,
		Action:   action,
		Mode:     mode,
		Status:   status,
		Operator: operator,
		Message:  message,
		Detail:   detail,
	}
}

func parseBackupFileID(value string) (uint, error) {
	id, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}
