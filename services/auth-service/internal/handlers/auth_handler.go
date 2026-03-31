package handlers

import (
	"cmdb-v2/pkg/common"
	"cmdb-v2/pkg/middleware"
	"cmdb-v2/pkg/utils"
	"cmdb-v2/services/auth-service/internal/models"
	"cmdb-v2/services/auth-service/internal/service"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthHandler struct {
	userService   *service.UserService
	systemService *service.SystemService
	avatarDir     string
}

func NewAuthHandler(us *service.UserService, ss *service.SystemService, avatarDir string) *AuthHandler {
	return &AuthHandler{
		userService:   us,
		systemService: ss,
		avatarDir:     avatarDir,
	}
}

// Login 处理登录请求
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	_, user, err := h.userService.Authenticate(req.Username, req.Password)
	if err != nil {
		common.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	permissions, err := h.systemService.ResolveRolePermissions(user.Role)
	if err != nil {
		common.Error(c, http.StatusInternalServerError, "计算权限失败: "+err.Error())
		return
	}

	token, err := utils.GenerateToken(user.ID, user.Username, user.Role, permissions)
	if err != nil {
		common.Error(c, http.StatusInternalServerError, "生成 Token 失败: "+err.Error())
		return
	}

	common.Success(c, models.LoginResponse{
		Token:       token,
		User:        models.ToUserProfile(user),
		Permissions: permissions,
	})
}

func (h *AuthHandler) GetMine(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		common.Error(c, http.StatusUnauthorized, "用户身份格式错误")
		return
	}

	user, err := h.userService.GetByID(userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			common.Error(c, http.StatusUnauthorized, "当前用户已失效，请重新登录")
			return
		}
		common.Error(c, http.StatusInternalServerError, "获取用户信息失败: "+err.Error())
		return
	}

	common.Success(c, models.ToUserProfile(user))
}

func (h *AuthHandler) UpdateMine(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		common.Error(c, http.StatusUnauthorized, "用户身份格式错误")
		return
	}

	var req models.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	user, err := h.userService.UpdateProfile(userID, req)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			common.Error(c, http.StatusUnauthorized, "用户会话已过期")
			return
		}
		common.Error(c, http.StatusInternalServerError, "更新用户信息失败: "+err.Error())
		return
	}

	common.Success(c, models.ToUserProfile(user))
}

func (h *AuthHandler) UpdateMyPassword(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		common.Error(c, http.StatusUnauthorized, "用户身份格式错误")
		return
	}

	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, http.StatusBadRequest, "请求参数校验失败: "+err.Error())
		return
	}

	if err := h.userService.UpdatePassword(userID, req); err != nil {
		common.Error(c, http.StatusBadRequest, "修改密码失败: "+err.Error())
		return
	}

	common.Success(c, gin.H{"message": "密码修改成功"})
}

func (h *AuthHandler) UploadMyAvatar(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		common.Error(c, http.StatusUnauthorized, "用户身份格式错误")
		return
	}

	user, err := h.userService.GetByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.Error(c, http.StatusUnauthorized, "当前用户已失效，请重新登录")
			return
		}
		common.Error(c, http.StatusInternalServerError, "获取用户信息失败: "+err.Error())
		return
	}

	avatarURL, err := h.saveAvatarFile(c, userID)
	if err != nil {
		common.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	updatedUser, err := h.userService.UpdateAvatar(userID, avatarURL)
	if err != nil {
		common.Error(c, http.StatusInternalServerError, "保存头像失败: "+err.Error())
		return
	}

	h.removeOldAvatarFile(user.Avatar, avatarURL)
	common.Success(c, gin.H{
		"avatar": avatarURL,
		"user":   models.ToUserProfile(updatedUser),
	})
}

func (h *AuthHandler) UploadUserAvatar(c *gin.Context) {
	if !middleware.HasPermission(c, "sys:user:update") {
		common.Error(c, http.StatusForbidden, "没有用户修改权限")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		common.Error(c, http.StatusBadRequest, "无效的用户ID")
		return
	}

	user, err := h.userService.GetByID(uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.Error(c, http.StatusNotFound, "用户不存在")
			return
		}
		common.Error(c, http.StatusInternalServerError, "获取用户信息失败: "+err.Error())
		return
	}

	avatarURL, err := h.saveAvatarFile(c, uint(id))
	if err != nil {
		common.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	updatedUser, err := h.userService.UpdateAvatar(uint(id), avatarURL)
	if err != nil {
		common.Error(c, http.StatusInternalServerError, "保存头像失败: "+err.Error())
		return
	}

	h.removeOldAvatarFile(user.Avatar, avatarURL)
	common.Success(c, gin.H{
		"avatar": avatarURL,
		"user":   models.ToUserProfile(updatedUser),
	})
}

func (h *AuthHandler) ListUsers(c *gin.Context) {
	if !middleware.HasPermission(c, "sys:user:list") {
		common.Error(c, http.StatusForbidden, "没有用户列表查看权限")
		return
	}

	var query models.UserQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		// 如果绑定失败，使用默认值而不是返回 400
		query.Page = 1
		query.PageSize = 10
	}

	users, total, err := h.userService.ListUsers(query)
	if err != nil {
		common.Error(c, http.StatusInternalServerError, "获取用户列表失败: "+err.Error())
		return
	}

	profiles := make([]models.UserProfile, len(users))
	for i, u := range users {
		profiles[i] = models.ToUserProfile(&u)
	}

	common.Success(c, gin.H{
		"list":        profiles,
		"total":       total,
		"page":        query.Page,
		"currentPage": query.Page,
		"pageSize":    query.PageSize,
	})
}

func (h *AuthHandler) CreateUser(c *gin.Context) {
	if !middleware.HasPermission(c, "sys:user:create") {
		common.Error(c, http.StatusForbidden, "没有用户创建权限")
		return
	}
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, http.StatusBadRequest, "请求参数格式错误: "+err.Error())
		return
	}

	user := models.User{
		Username:    req.Username,
		Password:    req.Password,
		Nickname:    req.Nickname,
		Email:       req.Email,
		Phone:       req.Phone,
		Role:        req.Role,
		DeptID:      req.DeptID,
		Status:      req.Status,
		Description: req.Description,
	}

	// 兼容前端 parentId 字段名
	if user.DeptID == 0 && req.ParentID != 0 {
		user.DeptID = req.ParentID
	}

	// 如果依然为 0，设置一个默认部门 ID 以防止外键报错
	if user.DeptID == 0 {
		var firstDept models.Dept
		h.systemService.DB.First(&firstDept) // systemHandler 已持有 systemService.DB
		if firstDept.ID != 0 {
			user.DeptID = firstDept.ID
		}
	}

	// 手动转换 Sex
	if req.Sex != nil {
		switch v := req.Sex.(type) {
		case float64:
			user.Sex = int(v)
		case string:
			if v != "" {
				s, _ := strconv.Atoi(v)
				user.Sex = s
			}
		}
	}

	if err := h.userService.CreateUser(&user); err != nil {
		common.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	common.Success(c, models.ToUserProfile(&user))
}

func (h *AuthHandler) UpdateUser(c *gin.Context) {
	if !middleware.HasPermission(c, "sys:user:update") {
		common.Error(c, http.StatusForbidden, "没有用户修改权限")
		return
	}
	var req models.CreateUserRequest // 复用 Request 结构体处理 sex 转换
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, http.StatusBadRequest, "请求参数格式错误: "+err.Error())
		return
	}

	idStr := c.Param("id")
	id, _ := strconv.ParseUint(idStr, 10, 64)

	user := models.User{
		ID:          uint(id),
		Username:    req.Username,
		Password:    req.Password,
		Nickname:    req.Nickname,
		Email:       req.Email,
		Phone:       req.Phone,
		Role:        req.Role,
		DeptID:      req.DeptID,
		Status:      req.Status,
		Description: req.Description,
	}

	// 手动转换 Sex
	if req.Sex != nil {
		switch v := req.Sex.(type) {
		case float64:
			user.Sex = int(v)
		case string:
			if v != "" {
				s, _ := strconv.Atoi(v)
				user.Sex = s
			}
		}
	}

	if err := h.userService.UpdateUser(&user); err != nil {
		common.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	common.Success(c, models.ToUserProfile(&user))
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	if !middleware.HasPermission(c, "sys:user:reset_password") {
		common.Error(c, http.StatusForbidden, "没有重置密码权限")
		return
	}

	idStr := c.Param("id")
	id, _ := strconv.ParseUint(idStr, 10, 64)

	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, http.StatusBadRequest, "密码必填")
		return
	}

	if err := h.userService.ResetPassword(uint(id), req.Password); err != nil {
		common.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	common.Success(c, gin.H{"message": "重置成功"})
}

func (h *AuthHandler) UpdateUserRole(c *gin.Context) {
	if !middleware.HasPermission(c, "sys:user:assign_role") {
		common.Error(c, http.StatusForbidden, "没有分配用户角色权限")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		common.Error(c, http.StatusBadRequest, "无效的用户ID")
		return
	}

	var req models.UpdateUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	if err := h.userService.UpdateUserRole(uint(id), req.Role); err != nil {
		common.Error(c, http.StatusInternalServerError, "更新用户角色失败: "+err.Error())
		return
	}

	common.Success(c, gin.H{"message": "更新成功"})
}

func (h *AuthHandler) DeleteUser(c *gin.Context) {
	if !middleware.HasPermission(c, "sys:user:delete") {
		common.Error(c, http.StatusForbidden, "没有用户删除权限")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		common.Error(c, http.StatusBadRequest, "无效的用户ID")
		return
	}

	if err := h.userService.DeleteUser(uint(id)); err != nil {
		common.Error(c, http.StatusInternalServerError, "删除用户失败: "+err.Error())
		return
	}

	common.Success(c, gin.H{"message": "删除成功"})
}

func (h *AuthHandler) GetUserRoleIDs(c *gin.Context) {
	if !middleware.HasPermission(c, "sys:user:assign_role") {
		common.Error(c, http.StatusForbidden, "没有查看用户角色权限")
		return
	}

	var req struct {
		UserID uint `json:"userId" form:"userId"`
	}
	if err := c.ShouldBind(&req); err != nil || req.UserID == 0 {
		common.Error(c, http.StatusBadRequest, "参数错误")
		return
	}

	user, err := h.userService.GetByID(req.UserID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			common.Error(c, http.StatusNotFound, "用户不存在")
			return
		}
		common.Error(c, http.StatusInternalServerError, "查询用户角色失败: "+err.Error())
		return
	}

	role, err := h.systemService.GetRoleByCode(user.Role)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			common.Success(c, []uint{})
			return
		}
		common.Error(c, http.StatusInternalServerError, "查询角色失败: "+err.Error())
		return
	}

	common.Success(c, []uint{role.ID})
}

func getCurrentUserID(c *gin.Context) (uint, bool) {
	rawUserID, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}

	switch value := rawUserID.(type) {
	case uint:
		return value, true
	case int:
		return uint(value), true
	case int64:
		return uint(value), true
	case float64:
		return uint(value), true
	case string:
		parsed, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return 0, false
		}
		return uint(parsed), true
	default:
		return 0, false
	}
}

func (h *AuthHandler) saveAvatarFile(c *gin.Context, userID uint) (string, error) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return "", errors.New("请选择要上传的头像文件")
	}

	if fileHeader.Size > 2*1024*1024 {
		return "", errors.New("头像文件不能超过 2MB")
	}

	src, err := fileHeader.Open()
	if err != nil {
		return "", errors.New("读取头像文件失败")
	}
	defer src.Close()

	head := make([]byte, 512)
	n, err := io.ReadFull(src, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", errors.New("解析头像文件失败")
	}

	contentType := http.DetectContentType(head[:n])
	ext := avatarExtension(contentType)
	if ext == "" {
		return "", errors.New("仅支持 PNG、JPG、WEBP、GIF 格式的头像")
	}

	if err := os.MkdirAll(h.avatarDir, 0o755); err != nil {
		return "", errors.New("初始化头像目录失败")
	}

	filename := "avatar-" + strconv.FormatUint(uint64(userID), 10) + "-" + strconv.FormatInt(time.Now().UnixNano(), 10) + ext
	dstPath := filepath.Join(h.avatarDir, filename)

	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return "", errors.New("重置头像文件流失败")
	}

	dst, err := os.Create(dstPath)
	if err != nil {
		return "", errors.New("创建头像文件失败")
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", errors.New("保存头像文件失败")
	}

	return "/api/v1/auth/avatars/" + filename, nil
}

func avatarExtension(contentType string) string {
	switch contentType {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ""
	}
}

func (h *AuthHandler) removeOldAvatarFile(previousAvatar, currentAvatar string) {
	if previousAvatar == "" || previousAvatar == currentAvatar {
		return
	}

	const avatarPrefix = "/api/v1/auth/avatars/"
	if !strings.HasPrefix(previousAvatar, avatarPrefix) {
		return
	}

	filename := filepath.Base(previousAvatar)
	if filename == "." || filename == string(filepath.Separator) {
		return
	}

	_ = os.Remove(filepath.Join(h.avatarDir, filename))
}
