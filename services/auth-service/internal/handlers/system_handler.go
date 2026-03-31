package handlers

import (
	"cmdb-v2/pkg/common"
	"cmdb-v2/pkg/middleware"
	"cmdb-v2/services/auth-service/internal/models"
	"cmdb-v2/services/auth-service/internal/service"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

type SystemHandler struct {
	systemService *service.SystemService
	auditService  *service.SystemService
}

func NewSystemHandler(ss *service.SystemService) *SystemHandler {
	return &SystemHandler{
		systemService: ss,
		auditService:  ss,
	}
}

func (h *SystemHandler) SetAuditService(as *service.SystemService) {
	h.auditService = as
}

// Dept Handlers
func (h *SystemHandler) ListDepts(c *gin.Context) {
	if !middleware.HasPermission(c, "sys:dept:list") {
		common.Error(c, http.StatusForbidden, "没有部门查看权限")
		return
	}
	depts, err := h.systemService.ListDepts()
	if err != nil {
		common.Error(c, http.StatusInternalServerError, "获取部门列表失败: "+err.Error())
		return
	}
	common.Success(c, depts)
}

func (h *SystemHandler) CreateDept(c *gin.Context) {
	if !middleware.HasPermission(c, "sys:dept:create") {
		common.Error(c, http.StatusForbidden, "没有部门创建权限")
		return
	}
	var dept models.Dept
	if err := c.ShouldBindJSON(&dept); err != nil {
		common.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}
	if err := h.systemService.CreateDept(&dept); err != nil {
		common.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	common.Success(c, dept)
}

func (h *SystemHandler) UpdateDept(c *gin.Context) {
	if !middleware.HasPermission(c, "sys:dept:update") {
		common.Error(c, http.StatusForbidden, "没有部门修改权限")
		return
	}
	var dept models.Dept
	if err := c.ShouldBindJSON(&dept); err != nil {
		common.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}
	idStr := c.Param("id")
	if idStr != "" {
		id, _ := strconv.Atoi(idStr)
		dept.ID = uint(id)
	}
	if err := h.systemService.UpdateDept(&dept); err != nil {
		common.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	common.Success(c, dept)
}

func (h *SystemHandler) DeleteDept(c *gin.Context) {
	if !middleware.HasPermission(c, "sys:dept:delete") {
		common.Error(c, http.StatusForbidden, "没有部门删除权限")
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.systemService.DeleteDept(uint(id)); err != nil {
		common.Error(c, http.StatusBadRequest, "删除失败")
		return
	}
	common.Success(c, nil)
}

func (h *SystemHandler) ListRoles(c *gin.Context) {
	if !middleware.HasPermission(c, "sys:role:list") {
		common.Error(c, http.StatusForbidden, "没有角色查看权限")
		return
	}
	roles, total, err := h.systemService.ListRoles()
	if err != nil {
		common.Error(c, http.StatusInternalServerError, "获取角色列表失败: "+err.Error())
		return
	}

	// 检查请求 Body 中是否有分页关键字（角色管理页面发出的请求通常会有分页参数）
	// 如果是空 Body 的 POST（下拉框常用方式），返回纯数组
	common.Success(c, roles)
	_ = total
}

func (h *SystemHandler) ListRolesForTable(c *gin.Context) {
	if !middleware.HasPermission(c, "sys:role:list") {
		common.Error(c, http.StatusForbidden, "没有角色查看权限")
		return
	}
	roles, total, err := h.systemService.ListRoles()
	if err != nil {
		common.Error(c, http.StatusInternalServerError, "获取角色列表失败: "+err.Error())
		return
	}
	common.Success(c, gin.H{
		"list":  roles,
		"total": total,
	})
}

func (h *SystemHandler) CreateRole(c *gin.Context) {
	if !middleware.HasPermission(c, "sys:role:create") {
		common.Error(c, http.StatusForbidden, "没有角色创建权限")
		return
	}
	var role models.Role
	if err := c.ShouldBindJSON(&role); err != nil {
		common.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}
	if err := h.systemService.CreateRole(&role); err != nil {
		common.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	common.Success(c, role)
}

func (h *SystemHandler) UpdateRole(c *gin.Context) {
	if !middleware.HasPermission(c, "sys:role:update") {
		common.Error(c, http.StatusForbidden, "没有角色修改权限")
		return
	}
	var role models.Role
	if err := c.ShouldBindJSON(&role); err != nil {
		common.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}
	idStr := c.Param("id")
	if idStr != "" {
		id, _ := strconv.Atoi(idStr)
		role.ID = uint(id)
	}
	if err := h.systemService.UpdateRole(&role); err != nil {
		common.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	common.Success(c, role)
}

func (h *SystemHandler) DeleteRole(c *gin.Context) {
	if !middleware.HasPermission(c, "sys:role:delete") {
		common.Error(c, http.StatusForbidden, "没有角色删除权限")
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.systemService.DeleteRole(uint(id)); err != nil {
		common.Error(c, http.StatusBadRequest, "删除失败")
		return
	}
	common.Success(c, nil)
}

// Menu Handlers
func (h *SystemHandler) ListMenus(c *gin.Context) {
	if !middleware.HasPermission(c, "sys:menu:list") {
		common.Error(c, http.StatusForbidden, "没有菜单查看权限")
		return
	}
	menus, err := h.systemService.ListMenus()
	if err != nil {
		common.Error(c, http.StatusInternalServerError, "获取菜单列表失败: "+err.Error())
		return
	}
	common.Success(c, menus)
}

func (h *SystemHandler) CreateMenu(c *gin.Context) {
	if !middleware.HasPermission(c, "sys:menu:create") {
		common.Error(c, http.StatusForbidden, "没有菜单创建权限")
		return
	}
	var menu models.Menu
	if err := c.ShouldBindJSON(&menu); err != nil {
		common.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}
	if err := h.systemService.CreateMenu(&menu); err != nil {
		common.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	common.Success(c, menu)
}

func (h *SystemHandler) UpdateMenu(c *gin.Context) {
	if !middleware.HasPermission(c, "sys:menu:update") {
		common.Error(c, http.StatusForbidden, "没有菜单修改权限")
		return
	}
	var menu models.Menu
	if err := c.ShouldBindJSON(&menu); err != nil {
		common.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}
	idStr := c.Param("id")
	if idStr != "" {
		id, _ := strconv.Atoi(idStr)
		menu.ID = uint(id)
	}
	if err := h.systemService.UpdateMenu(&menu); err != nil {
		common.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	common.Success(c, menu)
}

func (h *SystemHandler) DeleteMenu(c *gin.Context) {
	if !middleware.HasPermission(c, "sys:menu:delete") {
		common.Error(c, http.StatusForbidden, "没有菜单删除权限")
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.systemService.DeleteMenu(uint(id)); err != nil {
		common.Error(c, http.StatusBadRequest, "删除失败")
		return
	}
	common.Success(c, nil)
}

func (h *SystemHandler) ListAuditLogs(c *gin.Context) {
	if !middleware.HasAnyPermission(c, "sys:audit:list", "ops:logs:view") {
		common.Error(c, http.StatusForbidden, "没有审计日志查看权限")
		return
	}
	logs, total, err := h.auditService.ListAuditLogs()
	if err != nil {
		common.Error(c, http.StatusInternalServerError, "获取审计日志失败: "+err.Error())
		return
	}
	common.Success(c, gin.H{
		"list":  logs,
		"total": total,
	})
}

func (h *SystemHandler) GetRoleMenuIds(c *gin.Context) {
	if !middleware.HasPermission(c, "sys:role:assign_menu") {
		common.Error(c, http.StatusForbidden, "没有查看角色菜单权限")
		return
	}
	var req struct {
		ID uint `json:"id" form:"id"`
	}
	if err := c.ShouldBind(&req); err != nil {
		common.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	ids, err := h.systemService.GetRoleMenuIds(req.ID)
	if err != nil {
		common.Error(c, http.StatusInternalServerError, "查询失败")
		return
	}
	common.Success(c, ids)
}

func (h *SystemHandler) UpdateRoleMenus(c *gin.Context) {
	if !middleware.HasPermission(c, "sys:role:assign_menu") {
		common.Error(c, http.StatusForbidden, "没有分配角色菜单权限")
		return
	}
	var req struct {
		RoleID  uint   `json:"roleId"`
		MenuIDs []uint `json:"menuIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	if err := h.systemService.UpdateRoleMenus(req.RoleID, req.MenuIDs); err != nil {
		common.Error(c, http.StatusInternalServerError, "更新失败")
		return
	}
	common.Success(c, nil)
}
