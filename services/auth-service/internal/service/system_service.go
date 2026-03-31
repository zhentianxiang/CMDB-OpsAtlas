package service

import (
	"cmdb-v2/services/auth-service/internal/models"
	"errors"
	"gorm.io/gorm"
	"sort"
)

type SystemService struct {
	DB *gorm.DB
}

func NewSystemService(db *gorm.DB) *SystemService {
	return &SystemService{DB: db}
}

// ListDepts 获取部门列表
func (s *SystemService) ListDepts() ([]models.Dept, error) {
	var depts []models.Dept
	err := s.DB.Order("`sort` asc").Find(&depts).Error
	return depts, err
}

func (s *SystemService) CreateDept(dept *models.Dept) error {
	var count int64
	s.DB.Model(&models.Dept{}).Where("name = ? AND parent_id = ?", dept.Name, dept.ParentID).Count(&count)
	if count > 0 {
		return errors.New("同级部门下已存在该名称")
	}

	// 如果没有设置排序，自动设置为最大排序 + 1
	if dept.Sort == 0 {
		var maxSort int
		s.DB.Model(&models.Dept{}).Where("parent_id = ?", dept.ParentID).Select("max(sort)").Scan(&maxSort)
		dept.Sort = maxSort + 1
	}

	return s.DB.Create(dept).Error
}

func (s *SystemService) UpdateDept(dept *models.Dept) error {
	if dept.ID == 0 {
		return errors.New("无效的部门ID")
	}
	// 使用 Updates 避免 Save 导致的意外全字段覆盖（虽然此处用 Save 也可以，但明确 ID 更安全）
	return s.DB.Model(&models.Dept{}).Where("id = ?", dept.ID).Updates(dept).Error
}

func (s *SystemService) DeleteDept(id uint) error {
	return s.DB.Delete(&models.Dept{}, id).Error
}

// ListRoles 获取角色列表
func (s *SystemService) ListRoles() ([]models.Role, int64, error) {
	var roles []models.Role
	var total int64
	s.DB.Model(&models.Role{}).Count(&total)
	err := s.DB.Order("`sort` asc").Find(&roles).Error
	return roles, total, err
}

func (s *SystemService) CreateRole(role *models.Role) error {
	var count int64
	s.DB.Model(&models.Role{}).Where("code = ?", role.Code).Count(&count)
	if count > 0 {
		return errors.New("角色标识已存在")
	}
	return s.DB.Create(role).Error
}

func (s *SystemService) UpdateRole(role *models.Role) error {
	return s.DB.Save(role).Error
}

func (s *SystemService) DeleteRole(id uint) error {
	// 使用 Unscoped 进行物理删除，彻底从数据库移除角色
	return s.DB.Unscoped().Delete(&models.Role{}, id).Error
}

// ListMenus 获取菜单列表
func (s *SystemService) ListMenus() ([]models.Menu, error) {
	var menus []models.Menu
	err := s.DB.Order("`rank` asc").Find(&menus).Error
	return menus, err
}

func (s *SystemService) CreateMenu(menu *models.Menu) error {
	var count int64
	s.DB.Model(&models.Menu{}).Where("name = ? OR path = ?", menu.Name, menu.Path).Count(&count)
	if count > 0 {
		return errors.New("菜单名称或路由路径已存在")
	}
	return s.DB.Create(menu).Error
}

func (s *SystemService) UpdateMenu(menu *models.Menu) error {
	return s.DB.Save(menu).Error
}

func (s *SystemService) DeleteMenu(id uint) error {
	// 使用 Unscoped 进行物理删除，彻底从数据库移除菜单
	return s.DB.Unscoped().Delete(&models.Menu{}, id).Error
}

// ListAuditLogs 获取审计日志列表
func (s *SystemService) ListAuditLogs() ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64
	s.DB.Model(&models.AuditLog{}).Count(&total)
	err := s.DB.Order("id desc").Limit(100).Find(&logs).Error
	return logs, total, err
}

// GetRoleMenuIds 获取角色的菜单ID列表
func (s *SystemService) GetRoleMenuIds(roleID uint) ([]uint, error) {
	var menuIDs []uint
	err := s.DB.Model(&models.RoleMenu{}).Where("role_id = ?", roleID).Pluck("menu_id", &menuIDs).Error
	return menuIDs, err
}

// UpdateRoleMenus 更新角色的菜单关联
func (s *SystemService) UpdateRoleMenus(roleID uint, menuIDs []uint) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		// 先删除旧关联
		if err := tx.Where("role_id = ?", roleID).Delete(&models.RoleMenu{}).Error; err != nil {
			return err
		}
		// 批量插入新关联
		for _, mid := range menuIDs {
			if err := tx.Create(&models.RoleMenu{RoleID: roleID, MenuID: mid}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *SystemService) GetRoleByCode(code string) (*models.Role, error) {
	var role models.Role
	if err := s.DB.Where("code = ?", code).First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (s *SystemService) ResolveRolePermissions(roleCode string) ([]string, error) {
	if roleCode == "" {
		return []string{}, nil
	}
	if roleCode == "admin" {
		return []string{"*:*:*"}, nil
	}

	role, err := s.GetRoleByCode(roleCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []string{}, nil
		}
		return nil, err
	}

	var menus []models.Menu
	if err := s.DB.Model(&models.Menu{}).
		Joins("JOIN role_menus ON role_menus.menu_id = menus.id").
		Where("role_menus.role_id = ?", role.ID).
		Where("menus.auths <> ''").
		Find(&menus).Error; err != nil {
		return nil, err
	}

	uniq := make(map[string]struct{}, len(menus))
	for _, menu := range menus {
		if menu.Auths == "" {
			continue
		}
		uniq[menu.Auths] = struct{}{}
	}

	permissions := make([]string, 0, len(uniq))
	for item := range uniq {
		permissions = append(permissions, item)
	}
	sort.Strings(permissions)
	return permissions, nil
}
