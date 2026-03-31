package handlers

import (
	"bufio"
	"cmdb-v2/pkg/common"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type serviceLogResponse struct {
	ServiceName string   `json:"service_name"`
	Source      string   `json:"source"`
	FilePath    string   `json:"file_path"`
	Lines       []string `json:"lines"`
	LineCount   int      `json:"line_count"`
}

const defaultServiceLogDir = "/app/logs"

var opsServiceLogFiles = map[string]string{
	"auth-service":          "auth-service.log",
	"cluster-service":       "cluster-service.log",
	"host-service":          "host-service.log",
	"app-service":           "app-service.log",
	"port-service":          "port-service.log",
	"domain-service":        "domain-service.log",
	"dependency-service":    "dependency-service.log",
	"topology-service":      "topology-service.log",
	"cmdb-transfer-service": "cmdb-transfer-service.log",
}

func (h *Handler) GetServiceLogs(c *gin.Context) {
	serviceName := strings.TrimSpace(c.DefaultQuery("service", "cmdb-transfer-service"))
	logFileName, exists := opsServiceLogFiles[serviceName]
	if !exists {
		common.Error(c, http.StatusBadRequest, "不支持的服务名称")
		return
	}

	lineCount := parsePositiveInt(c.DefaultQuery("lines", "200"), 200, 20, 1000)
	sinceMinutes := parsePositiveInt(c.DefaultQuery("sinceMinutes", "30"), 30, 1, 24*60)
	sinceTime := time.Now().Add(-time.Duration(sinceMinutes) * time.Minute)

	logDir := os.Getenv("LOG_DIR")
	if logDir == "" {
		logDir = defaultServiceLogDir
	}

	logPath := filepath.Join(logDir, logFileName)
	lines, err := readRecentLogLines(logPath, lineCount, sinceTime)
	if err != nil {
		common.Error(c, http.StatusInternalServerError, "获取服务日志失败: "+err.Error())
		return
	}

	common.Success(c, serviceLogResponse{
		ServiceName: serviceName,
		Source:      "log file",
		FilePath:    logPath,
		Lines:       lines,
		LineCount:   len(lines),
	})
}

func readRecentLogLines(path string, lineCount int, sinceTime time.Time) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("日志文件不存在: %s", path)
		}
		return nil, err
	}
	defer file.Close()

	bufferSize := lineCount * 8
	if bufferSize < lineCount {
		bufferSize = lineCount
	}

	ring := make([]string, 0, bufferSize)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		ring = append(ring, line)
		if len(ring) > bufferSize {
			ring = ring[1:]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	filtered := make([]string, 0, len(ring))
	for _, line := range ring {
		lineTime, ok := parseLogTimestamp(line)
		if ok && lineTime.Before(sinceTime) {
			continue
		}
		filtered = append(filtered, line)
	}

	if len(filtered) > lineCount {
		filtered = filtered[len(filtered)-lineCount:]
	}

	return filtered, nil
}

func parseLogTimestamp(line string) (time.Time, bool) {
	candidates := []string{
		"2006/01/02 15:04:05.999999",
		"2006/01/02 15:04:05",
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05",
		time.RFC3339,
		time.RFC3339Nano,
	}

	for _, layout := range candidates {
		length := len(layout)
		if len(line) < length {
			continue
		}
		value := line[:length]
		parsed, err := time.ParseInLocation(layout, value, time.Local)
		if err == nil {
			return parsed, true
		}
	}

	return time.Time{}, false
}

func parsePositiveInt(raw string, fallback int, min int, max int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
