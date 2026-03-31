package models

import (
	"time"

	"gorm.io/gorm"
)
type Cluster struct {
	gorm.Model
	UserID uint   `json:"user_id" gorm:"index"`
	Name   string `json:"name" gorm:"size:128;not null;uniqueIndex"`
	Type   string `json:"type" gorm:"size:64"`
	Env    string `json:"env" gorm:"size:32"`
	Remark string `json:"remark" gorm:"size:255"`
}

type Host struct {
	gorm.Model
	UserID    uint   `json:"user_id" gorm:"index"`
	Name      string `json:"name" gorm:"size:128;not null"`
	IP        string `json:"ip" gorm:"size:64;uniqueIndex"`
	PublicIP  string `json:"public_ip" gorm:"size:64"`
	PrivateIP string `json:"private_ip" gorm:"size:64"`
	ClusterID *uint  `json:"cluster_id"`
	CPU       int    `json:"cpu"`
	Memory    int    `json:"memory"`
	OS        string `json:"os" gorm:"size:64"`
	Status    string `json:"status" gorm:"size:32"`
	Remark    string `json:"remark" gorm:"size:255"`
}

type App struct {
	gorm.Model
	UserID     uint   `json:"user_id" gorm:"index"`
	Name       string `json:"name" gorm:"size:128;not null"`
	HostID     uint   `json:"host_id" gorm:"index;not null"`
	Type       string `json:"type" gorm:"size:64"`
	Version    string `json:"version" gorm:"size:64"`
	DeployType string `json:"deploy_type" gorm:"size:64"`
	Remark     string `json:"remark" gorm:"size:255"`
}

type Port struct {
	gorm.Model
	UserID   uint   `json:"user_id" gorm:"index"`
	AppID    uint   `json:"app_id" gorm:"index;not null"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol" gorm:"size:16"`
	IsPublic bool   `json:"is_public"`
	Remark   string `json:"remark" gorm:"size:255"`
}

type Domain struct {
	gorm.Model
	UserID uint   `json:"user_id" gorm:"index"`
	Domain string `json:"domain" gorm:"size:255;index;not null"`
	AppID  *uint  `json:"app_id" gorm:"index"`
	HostID *uint  `json:"host_id" gorm:"index"`
	Remark string `json:"remark" gorm:"size:255"`
}

type Dependency struct {
	gorm.Model
	UserID       uint   `json:"user_id" gorm:"index"`
	SourceAppID  *uint  `json:"source_app_id" gorm:"index"`
	TargetAppID  *uint  `json:"target_app_id" gorm:"index"`
	SourceHostID *uint  `json:"source_host_id" gorm:"index"`
	TargetHostID *uint  `json:"target_host_id" gorm:"index"`
	DomainID     *uint  `json:"domain_id"`
	SourceNode   string `json:"source_node" gorm:"size:128"`
	TargetNode   string `json:"target_node" gorm:"size:128"`
	Desc         string `json:"desc" gorm:"size:255"`
	Remark       string `json:"remark" gorm:"size:255"`
}

type TransferRecord struct {
	gorm.Model
	UserID          uint   `json:"user_id" gorm:"index"`
	Action          string `json:"action" gorm:"size:32;index"`
	Mode            string `json:"mode" gorm:"size:32;index"`
	Status          string `json:"status" gorm:"size:32;index"`
	Filename        string `json:"filename" gorm:"size:255"`
	Operator        string `json:"operator" gorm:"size:64"`
	Message         string `json:"message" gorm:"size:255"`
	Detail          string `json:"detail" gorm:"type:text"`
	AddedSummary    string `json:"added_summary" gorm:"type:text"`
	SkippedSummary  string `json:"skipped_summary" gorm:"type:text"`
	CurrentSummary  string `json:"current_summary" gorm:"type:text"`
	IncomingSummary string `json:"incoming_summary" gorm:"type:text"`
}

type BackupPolicy struct {
	gorm.Model
	UserID        uint       `json:"user_id" gorm:"index"`
	Enabled       bool       `json:"enabled"`
	BackupHour    int        `json:"backup_hour"`
	RetentionDays int        `json:"retention_days"`
	BackupTypes   string     `json:"backup_types" gorm:"size:128"`
	BackupDir     string     `json:"backup_dir" gorm:"size:255"`
	LastRunAt     *time.Time `json:"last_run_at"`
}

type BackupFile struct {
	gorm.Model
	UserID        uint       `json:"user_id" gorm:"index"`
	BatchNo       string     `json:"batch_no" gorm:"size:64;index"`
	TriggerSource string     `json:"trigger_source" gorm:"size:32;index"`
	BackupType    string     `json:"backup_type" gorm:"size:32;index"`
	Status        string     `json:"status" gorm:"size:32;index"`
	Filename      string     `json:"filename" gorm:"size:255"`
	FilePath      string     `json:"file_path" gorm:"size:255"`
	SizeBytes     int64      `json:"size_bytes"`
	Message       string     `json:"message" gorm:"size:255"`
	Operator      string     `json:"operator" gorm:"size:64"`
	StartedAt     time.Time  `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at"`
	ExpiresAt     *time.Time `json:"expires_at"`
}

// AuditLog 全局审计日志
type AuditLog struct {
	gorm.Model
	UserID    uint   `json:"userId" gorm:"index"`
	Username  string `json:"username" gorm:"size:64"`
	Operation string `json:"operation" gorm:"size:128"` // 操作描述
	Method    string `json:"method" gorm:"size:16"`    // GET, POST...
	Path      string `json:"path" gorm:"size:255"`
	IP        string `json:"ip" gorm:"size:64"`
	Status    int    `json:"status"`                   // 响应状态码
	Duration  int64  `json:"duration"`                 // 耗时(ms)
	Payload   string `json:"payload" gorm:"type:text"` // 请求参数
}
