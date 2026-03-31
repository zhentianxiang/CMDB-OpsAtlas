package service

import (
	"cmdb-v2/services/auth-service/internal/models"
	"errors"
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ValidatePassword 校验密码复杂度: 至少8位, 包含英文和数字, 允许安全特殊字符 (@#$%.!&*-)
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("密码长度至少为 8 位")
	}

	// 允许：字母、数字、以及常见的安全特殊字符 . @ # $ % ^ & * ! _ -
	// 排除：' " \ ; ( ) 等可能对 SQL 或 Shell 产生潜在干扰的字符
	isValidChar := regexp.MustCompile(`^[a-zA-Z0-9.@#$%^&*!_-]+$`).MatchString(password)
	if !isValidChar {
		return errors.New("密码仅支持字母、数字及常用符号(.@#$%^&*!_-)")
	}

	// 必须包含字母
	hasLetter := regexp.MustCompile(`[a-zA-Z]`).MatchString(password)
	// 必须包含数字
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)

	if !hasLetter || !hasNumber {
		return errors.New("密码必须同时包含英文字母和数字")
	}

	return nil
}

type UserService struct {
	DB *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{DB: db}
}

// Authenticate 用户登录鉴权
func (s *UserService) Authenticate(username, password string) (string, *models.User, error) {
	var user models.User
	if err := s.DB.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, errors.New("用户名或密码错误")
		}
		return "", nil, err
	}

	if !verifyPassword(user.Password, password) {
		return "", nil, errors.New("用户名或密码错误")
	}

	if !isHashedPassword(user.Password) {
		hashedPassword, err := HashPassword(password)
		if err != nil {
			return "", nil, err
		}
		if err := s.DB.Model(&user).Update("password", hashedPassword).Error; err != nil {
			return "", nil, err
		}
		user.Password = hashedPassword
	}

	return "", &user, nil
}

func (s *UserService) GetByID(userID uint) (*models.User, error) {
	var user models.User
	if err := s.DB.Preload("Dept").First(&user, userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserService) UpdateProfile(userID uint, req models.UpdateProfileRequest) (*models.User, error) {
	user, err := s.GetByID(userID)
	if err != nil {
		return nil, err
	}

	user.Nickname = req.Nickname
	user.Email = req.Email
	user.Phone = req.Phone
	user.Avatar = req.Avatar
	user.Description = req.Description
	user.Sex = req.Sex
	if req.DeptID != 0 {
		user.DeptID = req.DeptID
	}

	if err := s.DB.Save(user).Error; err != nil {
		return nil, err
	}
	// 重新获取带部门信息的 User
	return s.GetByID(userID)
}

func (s *UserService) UpdatePassword(userID uint, req models.ChangePasswordRequest) error {
	user, err := s.GetByID(userID)
	if err != nil {
		return err
	}

	if req.NewPassword != req.ConfirmPassword {
		return errors.New("两次输入的新密码不一致")
	}

	if !verifyPassword(user.Password, req.OldPassword) {
		return errors.New("原密码错误")
	}

	if req.OldPassword == req.NewPassword {
		return errors.New("新密码不能与原密码相同")
	}

	// 密码复杂度校验
	if err := ValidatePassword(req.NewPassword); err != nil {
		return err
	}

	hashedPassword, err := HashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	return s.DB.Model(user).Update("password", hashedPassword).Error
}

func (s *UserService) UpdateAvatar(userID uint, avatar string) (*models.User, error) {
	if err := s.DB.Model(&models.User{}).Where("id = ?", userID).Update("avatar", avatar).Error; err != nil {
		return nil, err
	}
	return s.GetByID(userID)
}

func (s *UserService) CreateUser(user *models.User) error {
	// 1. 检查用户名是否已存在 (包括软删除的用户)
	var count int64
	s.DB.Unscoped().Model(&models.User{}).Where("username = ?", user.Username).Count(&count)
	if count > 0 {
		return errors.New("用户名 [" + user.Username + "] 已存在，请更换或联系管理员")
	}

	// 2. 密码校验及加密
	if user.Password == "" {
		user.Password = "admin123" // 设置初始默认密码
	}

	if err := ValidatePassword(user.Password); err != nil {
		return err
	}

	hashedPassword, err := HashPassword(user.Password)
	if err != nil {
		return err
	}
	user.Password = hashedPassword

	// 3. 设置默认角色
	if user.Role == "" {
		user.Role = "common"
	}

	return s.DB.Create(user).Error
}

func (s *UserService) UpdateUser(user *models.User) error {
	updates := make(map[string]interface{})
	if user.Nickname != "" {
		updates["nickname"] = user.Nickname
	}
	if user.Email != "" {
		updates["email"] = user.Email
	}
	if user.Phone != "" {
		updates["phone"] = user.Phone
	}
	if user.Avatar != "" {
		updates["avatar"] = user.Avatar
	}
	if user.Description != "" {
		updates["description"] = user.Description
	}
	if user.Role != "" {
		updates["role"] = user.Role
	}
	if user.DeptID != 0 {
		updates["dept_id"] = user.DeptID
	}
	updates["status"] = user.Status
	updates["sex"] = user.Sex

	if user.Password != "" {
		// 密码复杂度校验
		if err := ValidatePassword(user.Password); err != nil {
			return err
		}
		hashedPassword, err := HashPassword(user.Password)
		if err != nil {
			return err
		}
		updates["password"] = hashedPassword
	}

	return s.DB.Model(&models.User{}).Where("id = ?", user.ID).Updates(updates).Error
}

func (s *UserService) ResetPassword(userID uint, newPassword string) error {
	hashedPassword, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.DB.Model(&models.User{}).Where("id = ?", userID).Update("password", hashedPassword).Error
}

func (s *UserService) ListUsers(query models.UserQuery) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	db := s.DB.Model(&models.User{})
	if query.Username != "" {
		db = db.Where("username LIKE ?", "%"+query.Username+"%")
	}
	if query.Phone != "" {
		db = db.Where("phone LIKE ?", "%"+query.Phone+"%")
	}
	if query.Status != nil {
		db = db.Where("status = ?", *query.Status)
	}
	if query.DeptID != 0 {
		db = db.Where("dept_id = ?", query.DeptID)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 10
	}

	offset := (query.Page - 1) * query.PageSize
	if err := db.Preload("Dept").Limit(query.PageSize).Offset(offset).Order("id DESC").Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (s *UserService) UpdateUserRole(userID uint, role string) error {
	return s.DB.Model(&models.User{}).Where("id = ?", userID).Update("role", role).Error
}

func (s *UserService) DeleteUser(userID uint) error {
	// 使用 Unscoped 进行物理删除，避免软删除导致的唯一索引冲突
	return s.DB.Unscoped().Delete(&models.User{}, userID).Error
}

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func verifyPassword(storedPassword, password string) bool {
	if isHashedPassword(storedPassword) {
		return bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password)) == nil
	}
	return storedPassword == password
}

func isHashedPassword(password string) bool {
	return strings.HasPrefix(password, "$2a$") || strings.HasPrefix(password, "$2b$") || strings.HasPrefix(password, "$2y$")
}
