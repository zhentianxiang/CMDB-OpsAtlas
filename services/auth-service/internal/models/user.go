package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"createTime"`
	UpdatedAt   time.Time      `json:"updateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Username    string         `gorm:"uniqueIndex;size:64" json:"username"`
	Password    string         `gorm:"size:255" json:"password,omitempty"`
	Nickname    string         `gorm:"size:64" json:"nickname"`
	Email       string         `gorm:"size:128" json:"email"`
	Phone       string         `gorm:"size:32" json:"phone"`
	Avatar      string         `gorm:"size:255" json:"avatar"`
	Description string         `gorm:"size:255" json:"description"`
	Role        string         `gorm:"size:32" json:"role"`
	DeptID      uint           `json:"deptId"`
	Dept        *Dept          `gorm:"foreignKey:DeptID" json:"dept"`
	Sex         int            `json:"sex"`
	Status      int            `json:"status" gorm:"default:1"`
}

// CreateUserRequest 用于接收前端的新增用户请求，兼容 sex 为空字符串等情况
type CreateUserRequest struct {
	Username    string      `json:"username" binding:"required"`
	Password    string      `json:"password"`
	Nickname    string      `json:"nickname"`
	Email       string      `json:"email"`
	Phone       string      `json:"phone"`
	Role        string      `json:"role"`
	DeptID      uint        `json:"deptId"`
	ParentID    uint        `json:"parentId"` // 增加此字段以兼容前端表单
	Sex         interface{} `json:"sex"`
	Status      int         `json:"status"`
	Description string      `json:"description"`
}

type UserProfile struct {
	ID          uint      `json:"id"`
	Username    string    `json:"username"`
	Nickname    string    `json:"nickname"`
	Email       string    `json:"email"`
	Phone       string    `json:"phone"`
	Avatar      string    `json:"avatar"`
	Description string    `json:"description"`
	Role        string    `json:"role"`
	DeptID      uint      `json:"deptId"`
	Dept        *Dept     `json:"dept"`
	Sex         int       `json:"sex"`
	Status      int       `json:"status"`
	CreatedAt   time.Time `json:"createTime"`
	UpdatedAt   time.Time `json:"updateTime"`
}

func ToUserProfile(user *User) UserProfile {
	return UserProfile{
		ID:          user.ID,
		Username:    user.Username,
		Nickname:    user.Nickname,
		Email:       user.Email,
		Phone:       user.Phone,
		Avatar:      user.Avatar,
		Description: user.Description, // 对应前端的 "简介" 和 "备注"
		Role:        user.Role,
		DeptID:      user.DeptID,
		Dept:        user.Dept,
		Sex:         user.Sex,
		Status:      user.Status,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3"`
	Password string `json:"password" binding:"required,min=6"`
}

type UpdateUserRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

type UserQuery struct {
	Page     int    `form:"page" json:"page"`
	PageSize int    `form:"pageSize" json:"pageSize"`
	Username string `form:"username" json:"username"`
	Phone    string `form:"phone" json:"phone"`
	Status   *int   `form:"status" json:"status"`
	DeptID   uint   `form:"deptId" json:"deptId"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token       string      `json:"token"`
	User        UserProfile `json:"user"`
	Permissions []string    `json:"permissions"`
}

type UpdateProfileRequest struct {
	Nickname    string `json:"nickname"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Avatar      string `json:"avatar"`
	Description string `json:"description"`
	Sex         int    `json:"sex"`
	DeptID      uint   `json:"deptId"`
}

type ChangePasswordRequest struct {
	OldPassword     string `json:"oldPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required,min=6"`
	ConfirmPassword string `json:"confirmPassword" binding:"required"`
}
