package service

import (
	"cmdb-v2/services/auth-service/internal/models"
	"sort"

	"gorm.io/gorm"
)

type seedMenu struct {
	Key        string
	ParentKey  string
	MenuType   int
	Title      string
	Name       string
	Path       string
	Component  string
	Icon       string
	Rank       int
	Auths      string
	ShowLink   bool
	ShowParent bool
}

func SeedRBACData(db *gorm.DB) error {
	menus := defaultSeedMenus()
	menuIDs := make(map[string]uint, len(menus))

	for _, item := range menus {
		menu := models.Menu{
			MenuType:   item.MenuType,
			Title:      item.Title,
			Name:       item.Name,
			Path:       item.Path,
			Component:  item.Component,
			Icon:       item.Icon,
			Rank:       item.Rank,
			Auths:      item.Auths,
			ShowLink:   item.ShowLink,
			ShowParent: item.ShowParent,
		}

		if item.ParentKey != "" {
			menu.ParentID = menuIDs[item.ParentKey]
		}

		var existing models.Menu
		err := db.Where("name = ? OR (path <> '' AND path = ?)", item.Name, item.Path).First(&existing).Error
		if err == nil {
			existing.ParentID = menu.ParentID
			existing.MenuType = menu.MenuType
			existing.Title = menu.Title
			existing.Name = menu.Name
			existing.Path = menu.Path
			existing.Component = menu.Component
			existing.Icon = menu.Icon
			existing.Rank = menu.Rank
			existing.Auths = menu.Auths
			existing.ShowLink = menu.ShowLink
			existing.ShowParent = menu.ShowParent
			if saveErr := db.Save(&existing).Error; saveErr != nil {
				return saveErr
			}
			menuIDs[item.Key] = existing.ID
			continue
		}
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}

		if createErr := db.Create(&menu).Error; createErr != nil {
			return createErr
		}
		menuIDs[item.Key] = menu.ID
	}

	return seedRoleMenus(db, menuIDs)
}

func seedRoleMenus(db *gorm.DB, menuIDs map[string]uint) error {
	var roles []models.Role
	if err := db.Find(&roles).Error; err != nil {
		return err
	}

	roleMenus := map[string][]string{
		"admin": allMenuKeys(menuIDs),
		"asset_mgr": {
			"cmdb_root", "cmdb_resources",
			"ops_root", "ops_runtime", "ops_tasks", "ops_backup",
			"cmdb_cluster_create", "cmdb_cluster_update", "cmdb_cluster_delete",
			"cmdb_host_create", "cmdb_host_update", "cmdb_host_delete",
			"cmdb_app_create", "cmdb_app_update", "cmdb_app_delete",
			"cmdb_port_create", "cmdb_port_update", "cmdb_port_delete",
			"cmdb_domain_create", "cmdb_domain_update", "cmdb_domain_delete",
			"cmdb_dependency_create", "cmdb_dependency_update", "cmdb_dependency_delete",
			"cmdb_export", "cmdb_import",
			"ops_backup_manage", "ops_logs",
			"ops_sql_read",
		},
		"dev_ops": {
			"cmdb_root", "cmdb_resources",
			"ops_root", "ops_runtime", "ops_tasks", "ops_logs",
			"cmdb_app_create", "cmdb_app_update",
			"cmdb_port_create", "cmdb_port_update",
			"cmdb_domain_create", "cmdb_domain_update",
			"cmdb_dependency_create", "cmdb_dependency_update",
			"cmdb_export",
			"ops_sql_read",
		},
		"auditor": {
			"cmdb_root", "cmdb_resources",
			"ops_root", "ops_runtime", "ops_logs", "ops_tasks", "ops_backup",
			"system_root", "system_audit",
		},
		"common": {
			"cmdb_root", "cmdb_resources",
			"ops_root", "ops_runtime",
		},
	}

	for _, role := range roles {
		keys, ok := roleMenus[role.Code]
		if !ok {
			continue
		}

		var existingMenus []uint
		if err := db.Model(&models.RoleMenu{}).Where("role_id = ?", role.ID).Pluck("menu_id", &existingMenus).Error; err != nil {
			return err
		}

		existingSet := make(map[uint]struct{}, len(existingMenus))
		for _, menuID := range existingMenus {
			existingSet[menuID] = struct{}{}
		}
		items := make([]models.RoleMenu, 0, len(keys))
		for _, key := range keys {
			menuID, exists := menuIDs[key]
			if !exists || menuID == 0 {
				continue
			}
			if _, exists := existingSet[menuID]; exists {
				continue
			}
			existingSet[menuID] = struct{}{}
			items = append(items, models.RoleMenu{RoleID: role.ID, MenuID: menuID})
		}
		if len(items) == 0 {
			continue
		}
		if err := db.Create(&items).Error; err != nil {
			return err
		}
	}

	return nil
}

func allMenuKeys(menuIDs map[string]uint) []string {
	keys := make([]string, 0, len(menuIDs))
	for key := range menuIDs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func defaultSeedMenus() []seedMenu {
	return []seedMenu{
		{Key: "cmdb_root", MenuType: 0, Title: "CMDB", Name: "CMDB", Path: "/cmdb", Component: "Layout", Icon: "ri:database-2-line", Rank: 1, Auths: "cmdb:view", ShowLink: true, ShowParent: true},
		{Key: "cmdb_resources", ParentKey: "cmdb_root", MenuType: 1, Title: "CMDB 资源", Name: "CMDBResources", Path: "/cmdb/resources", Component: "/views/cmdb/index.vue", Auths: "cmdb:view", ShowLink: true, ShowParent: true},
		{Key: "cmdb_cluster_create", ParentKey: "cmdb_resources", MenuType: 3, Title: "新增集群", Name: "CmdbClusterCreate", Auths: "cmdb:cluster:create", ShowLink: false},
		{Key: "cmdb_cluster_update", ParentKey: "cmdb_resources", MenuType: 3, Title: "编辑集群", Name: "CmdbClusterUpdate", Auths: "cmdb:cluster:update", ShowLink: false},
		{Key: "cmdb_cluster_delete", ParentKey: "cmdb_resources", MenuType: 3, Title: "删除集群", Name: "CmdbClusterDelete", Auths: "cmdb:cluster:delete", ShowLink: false},
		{Key: "cmdb_host_create", ParentKey: "cmdb_resources", MenuType: 3, Title: "新增主机", Name: "CmdbHostCreate", Auths: "cmdb:host:create", ShowLink: false},
		{Key: "cmdb_host_update", ParentKey: "cmdb_resources", MenuType: 3, Title: "编辑主机", Name: "CmdbHostUpdate", Auths: "cmdb:host:update", ShowLink: false},
		{Key: "cmdb_host_delete", ParentKey: "cmdb_resources", MenuType: 3, Title: "删除主机", Name: "CmdbHostDelete", Auths: "cmdb:host:delete", ShowLink: false},
		{Key: "cmdb_app_create", ParentKey: "cmdb_resources", MenuType: 3, Title: "新增应用", Name: "CmdbAppCreate", Auths: "cmdb:app:create", ShowLink: false},
		{Key: "cmdb_app_update", ParentKey: "cmdb_resources", MenuType: 3, Title: "编辑应用", Name: "CmdbAppUpdate", Auths: "cmdb:app:update", ShowLink: false},
		{Key: "cmdb_app_delete", ParentKey: "cmdb_resources", MenuType: 3, Title: "删除应用", Name: "CmdbAppDelete", Auths: "cmdb:app:delete", ShowLink: false},
		{Key: "cmdb_port_create", ParentKey: "cmdb_resources", MenuType: 3, Title: "新增端口", Name: "CmdbPortCreate", Auths: "cmdb:port:create", ShowLink: false},
		{Key: "cmdb_port_update", ParentKey: "cmdb_resources", MenuType: 3, Title: "编辑端口", Name: "CmdbPortUpdate", Auths: "cmdb:port:update", ShowLink: false},
		{Key: "cmdb_port_delete", ParentKey: "cmdb_resources", MenuType: 3, Title: "删除端口", Name: "CmdbPortDelete", Auths: "cmdb:port:delete", ShowLink: false},
		{Key: "cmdb_domain_create", ParentKey: "cmdb_resources", MenuType: 3, Title: "新增域名", Name: "CmdbDomainCreate", Auths: "cmdb:domain:create", ShowLink: false},
		{Key: "cmdb_domain_update", ParentKey: "cmdb_resources", MenuType: 3, Title: "编辑域名", Name: "CmdbDomainUpdate", Auths: "cmdb:domain:update", ShowLink: false},
		{Key: "cmdb_domain_delete", ParentKey: "cmdb_resources", MenuType: 3, Title: "删除域名", Name: "CmdbDomainDelete", Auths: "cmdb:domain:delete", ShowLink: false},
		{Key: "cmdb_dependency_create", ParentKey: "cmdb_resources", MenuType: 3, Title: "新增依赖", Name: "CmdbDependencyCreate", Auths: "cmdb:dependency:create", ShowLink: false},
		{Key: "cmdb_dependency_update", ParentKey: "cmdb_resources", MenuType: 3, Title: "编辑依赖", Name: "CmdbDependencyUpdate", Auths: "cmdb:dependency:update", ShowLink: false},
		{Key: "cmdb_dependency_delete", ParentKey: "cmdb_resources", MenuType: 3, Title: "删除依赖", Name: "CmdbDependencyDelete", Auths: "cmdb:dependency:delete", ShowLink: false},
		{Key: "ops_root", MenuType: 0, Title: "运维管理", Name: "OpsManagement", Path: "/ops", Component: "Layout", Icon: "ri:tools-line", Rank: 2, Auths: "ops:view", ShowLink: true, ShowParent: true},
		{Key: "ops_runtime", ParentKey: "ops_root", MenuType: 1, Title: "运行状态", Name: "CMDBOpsRuntime", Path: "/ops/runtime", Component: "/views/cmdb/ops/runtime.vue", Auths: "ops:runtime:view", ShowLink: true, ShowParent: true},
		{Key: "ops_logs", ParentKey: "ops_root", MenuType: 1, Title: "日志中心", Name: "OpsLogs", Path: "/ops/logs", Component: "/views/ops/logs.vue", Auths: "ops:logs:view", ShowLink: true, ShowParent: true},
		{Key: "ops_tasks", ParentKey: "ops_root", MenuType: 1, Title: "任务中心", Name: "OpsTasks", Path: "/ops/tasks", Component: "/views/ops/tasks.vue", Auths: "ops:task:view", ShowLink: true, ShowParent: true},
		{Key: "ops_backup", ParentKey: "ops_root", MenuType: 1, Title: "备份恢复", Name: "CMDBOpsBackup", Path: "/ops/backup", Component: "/views/cmdb/ops/backup.vue", Auths: "ops:backup:view", ShowLink: true, ShowParent: true},
		{Key: "ops_sql", ParentKey: "ops_root", MenuType: 1, Title: "SQL 控制台", Name: "OpsSqlConsole", Path: "/ops/sql", Component: "/views/ops/sql.vue", Auths: "ops:sql:read", ShowLink: true, ShowParent: true},
		{Key: "cmdb_export", ParentKey: "ops_backup", MenuType: 3, Title: "导出 CMDB", Name: "CmdbExport", Auths: "cmdb:export", ShowLink: false},
		{Key: "cmdb_import", ParentKey: "ops_backup", MenuType: 3, Title: "导入 CMDB", Name: "CmdbImport", Auths: "cmdb:import", ShowLink: false},
		{Key: "ops_backup_manage", ParentKey: "ops_backup", MenuType: 3, Title: "备份管理", Name: "CmdbBackupManage", Auths: "ops:backup:manage", ShowLink: false},
		{Key: "ops_sql_read", ParentKey: "ops_sql", MenuType: 3, Title: "只读 SQL", Name: "OpsSqlRead", Auths: "ops:sql:read", ShowLink: false},
		{Key: "ops_sql_execute", ParentKey: "ops_sql", MenuType: 3, Title: "执行 SQL", Name: "OpsSqlExecute", Auths: "ops:sql:execute", ShowLink: false},

		{Key: "system_root", MenuType: 0, Title: "系统管理", Name: "System", Path: "/system", Component: "Layout", Icon: "ri:settings-3-line", Rank: 10, Auths: "sys:manage:view", ShowLink: true, ShowParent: true},
		{Key: "system_user", ParentKey: "system_root", MenuType: 1, Title: "用户管理", Name: "SystemUser", Path: "/system/user/index", Component: "/views/system/user/index.vue", Auths: "sys:user:list", ShowLink: true, ShowParent: true},
		{Key: "system_role", ParentKey: "system_root", MenuType: 1, Title: "角色管理", Name: "SystemRole", Path: "/system/role/index", Component: "/views/system/role/index.vue", Auths: "sys:role:list", ShowLink: true, ShowParent: true},
		{Key: "system_menu", ParentKey: "system_root", MenuType: 1, Title: "菜单管理", Name: "SystemMenu", Path: "/system/menu/index", Component: "/views/system/menu/index.vue", Auths: "sys:menu:list", ShowLink: true, ShowParent: true},
		{Key: "system_dept", ParentKey: "system_root", MenuType: 1, Title: "部门管理", Name: "SystemDept", Path: "/system/dept/index", Component: "/views/system/dept/index.vue", Auths: "sys:dept:list", ShowLink: true, ShowParent: true},
		{Key: "system_audit", ParentKey: "system_root", MenuType: 1, Title: "操作审计", Name: "SystemAudit", Path: "/system/audit/index", Component: "/views/system/audit/index.vue", Auths: "sys:audit:list", ShowLink: true, ShowParent: true},
		{Key: "system_user_create", ParentKey: "system_user", MenuType: 3, Title: "新增用户", Name: "SystemUserCreate", Auths: "sys:user:create", ShowLink: false},
		{Key: "system_user_update", ParentKey: "system_user", MenuType: 3, Title: "修改用户", Name: "SystemUserUpdate", Auths: "sys:user:update", ShowLink: false},
		{Key: "system_user_delete", ParentKey: "system_user", MenuType: 3, Title: "删除用户", Name: "SystemUserDelete", Auths: "sys:user:delete", ShowLink: false},
		{Key: "system_user_reset_password", ParentKey: "system_user", MenuType: 3, Title: "重置密码", Name: "SystemUserResetPassword", Auths: "sys:user:reset_password", ShowLink: false},
		{Key: "system_user_assign_role", ParentKey: "system_user", MenuType: 3, Title: "分配角色", Name: "SystemUserAssignRole", Auths: "sys:user:assign_role", ShowLink: false},
		{Key: "system_role_create", ParentKey: "system_role", MenuType: 3, Title: "新增角色", Name: "SystemRoleCreate", Auths: "sys:role:create", ShowLink: false},
		{Key: "system_role_update", ParentKey: "system_role", MenuType: 3, Title: "修改角色", Name: "SystemRoleUpdate", Auths: "sys:role:update", ShowLink: false},
		{Key: "system_role_delete", ParentKey: "system_role", MenuType: 3, Title: "删除角色", Name: "SystemRoleDelete", Auths: "sys:role:delete", ShowLink: false},
		{Key: "system_role_assign_menu", ParentKey: "system_role", MenuType: 3, Title: "分配菜单", Name: "SystemRoleAssignMenu", Auths: "sys:role:assign_menu", ShowLink: false},
		{Key: "system_menu_create", ParentKey: "system_menu", MenuType: 3, Title: "新增菜单", Name: "SystemMenuCreate", Auths: "sys:menu:create", ShowLink: false},
		{Key: "system_menu_update", ParentKey: "system_menu", MenuType: 3, Title: "修改菜单", Name: "SystemMenuUpdate", Auths: "sys:menu:update", ShowLink: false},
		{Key: "system_menu_delete", ParentKey: "system_menu", MenuType: 3, Title: "删除菜单", Name: "SystemMenuDelete", Auths: "sys:menu:delete", ShowLink: false},
		{Key: "system_dept_create", ParentKey: "system_dept", MenuType: 3, Title: "新增部门", Name: "SystemDeptCreate", Auths: "sys:dept:create", ShowLink: false},
		{Key: "system_dept_update", ParentKey: "system_dept", MenuType: 3, Title: "修改部门", Name: "SystemDeptUpdate", Auths: "sys:dept:update", ShowLink: false},
		{Key: "system_dept_delete", ParentKey: "system_dept", MenuType: 3, Title: "删除部门", Name: "SystemDeptDelete", Auths: "sys:dept:delete", ShowLink: false},
	}
}
