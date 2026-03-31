package models

import (
	"time"
	"gorm.io/gorm"
)

// Dept 部门模型
type Dept struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"createTime"`
	UpdatedAt time.Time      `json:"updateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Name      string         `json:"name" gorm:"size:128;not null"`
	ParentID  uint           `json:"parentId"`
	Sort      int            `json:"sort"`
	Principal string         `json:"principal" gorm:"size:64"`
	Phone     string         `json:"phone" gorm:"size:32"`
	Email     string         `json:"email" gorm:"size:128"`
	Status    int            `json:"status" gorm:"default:1"` // 1: 启用, 0: 停用
	Remark    string         `json:"remark" gorm:"size:255"`
}

// Role 角色模型
type Role struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"createTime"`
	UpdatedAt time.Time      `json:"updateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Name      string         `json:"name" gorm:"size:64;not null"`
	Code      string         `json:"code" gorm:"uniqueIndex;size:64;not null"` // 如 admin, common
	Sort      int            `json:"sort"`
	Status    int            `json:"status" gorm:"default:1"`
	Remark    string         `json:"remark" gorm:"size:255"`
}

// Menu 菜单模型
type Menu struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time      `json:"createTime"`
	UpdatedAt    time.Time      `json:"updateTime"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	ParentID     uint           `json:"parentId"`
	MenuType     int            `json:"menuType"` // 0: 目录, 1: 菜单, 2: 按钮
	Title        string         `json:"title" gorm:"size:128"`
	Name         string         `json:"name" gorm:"size:128"`
	Path         string         `json:"path" gorm:"size:255"`
	Component    string         `json:"component" gorm:"size:255"`
	Rank         int            `json:"rank"`
	Redirect     string         `json:"redirect" gorm:"size:255"`
	Icon         string         `json:"icon" gorm:"size:128"`
	ExtraIcon    string         `json:"extraIcon" gorm:"size:128"`
	EnterAnima   string         `json:"enterAnima" gorm:"size:64"`
	LeaveAnima   string         `json:"leaveAnima" gorm:"size:64"`
	ActivePath   string         `json:"activePath" gorm:"size:255"`
	Auths        string         `json:"auths" gorm:"size:255"` // 权限标识
	FrameSrc     string         `json:"frameSrc" gorm:"size:255"`
	FrameLoading bool           `json:"frameLoading"`
	KeepAlive    bool           `json:"keepAlive"`
	HiddenTag    bool           `json:"hiddenTag"`
	FixedTag     bool           `json:"fixedTag"`
	ShowLink     bool           `json:"showLink" gorm:"default:true"`
	ShowParent   bool           `json:"showParent" gorm:"default:true"`
}

// AuditLog 审计日志模型
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

// RoleMenu 角色-菜单关联模型
type RoleMenu struct {
	RoleID uint `gorm:"primaryKey"`
	MenuID uint `gorm:"primaryKey"`
}
