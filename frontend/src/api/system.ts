import { http } from "@/utils/http";

type Result = {
  code: number;
  message: string;
  data?: any;
};

type ResultTable = {
  code: number;
  message: string;
  data?: {
    /** 列表数据 */
    list: Array<any>;
    /** 总条目数 */
    total?: number;
    /** 每页显示条目个数 */
    pageSize?: number;
    /** 当前页数 */
    currentPage?: number;
  };
};

function compactParams(params?: Record<string, any>) {
  if (!params) return params;
  return Object.fromEntries(
    Object.entries(params).filter(([, value]) => value !== "" && value !== null && value !== undefined)
  );
}

/** 获取系统管理-用户管理列表 */
export const getUserList = (params?: object) => {
  return http.request<ResultTable>("get", "/api/v1/auth/users", {
    params: compactParams(params as Record<string, any>)
  });
};

/** 新增用户 */
export const createUser = (data?: object) => {
  return http.request<Result>("post", "/api/v1/auth/users", { data });
};

/** 修改用户 */
export const updateUser = (id: number, data?: object) => {
  return http.request<Result>("put", `/api/v1/auth/users/${id}`, { data });
};

/** 系统管理-用户管理-获取所有角色列表 */
export const getAllRoleList = () => {
  return http.request<Result>("post", "/api/v1/auth/role");
};

/** 系统管理-用户管理-修改用户角色 */
export const updateUserRole = (id: number, role: string) => {
  return http.request<Result>("put", `/api/v1/auth/users/${id}/role`, {
    data: { role }
  });
};

/** 系统管理-用户管理-删除用户 */
export const deleteUser = (id: number) => {
  return http.request<Result>("delete", `/api/v1/auth/users/${id}`);
};

/** 系统管理-用户管理-根据userId，获取对应角色id列表（userId：用户id） */
export const getRoleIds = (data?: object) => {
  return http.request<Result>("post", "/api/v1/auth/list-role-ids", { data });
};

export const resetUserPassword = (id: number, password: string) => {
  return http.request<Result>("put", `/api/v1/auth/users/${id}/password`, {
    data: { password }
  });
};

export const uploadUserAvatar = (id: number, file: Blob) => {
  const formData = new FormData();
  formData.append("file", file, "avatar.png");
  return http.request<Result>("post", `/api/v1/auth/users/${id}/avatar`, {
    data: formData
  });
};

/** 获取系统管理-部门管理列表 */
export const getDeptList = (params?: object) => {
  return http.request<Result>("get", "/api/v1/auth/dept", { params });
};

/** 新增部门 */
export const createDept = (data?: object) => {
  return http.request<Result>("post", "/api/v1/auth/dept", { data });
};

/** 修改部门 */
export const updateDept = (id: number, data?: object) => {
  return http.request<Result>("put", `/api/v1/auth/dept/${id}`, { data });
};

/** 删除部门 */
export const deleteDept = (id: number) => {
  return http.request<Result>("delete", `/api/v1/auth/dept/${id}`);
};

/** 获取系统管理-角色管理列表 */
export const getRoleList = (data?: object) => {
  return http.request<ResultTable>("post", "/api/v1/auth/roles", { data });
};

/** 新增角色 */
export const createRole = (data?: object) => {
  return http.request<Result>("post", "/api/v1/auth/role/create", { data });
};

/** 修改角色 */
export const updateRole = (id: number, data?: object) => {
  return http.request<Result>("put", `/api/v1/auth/role/${id}`, { data });
};

/** 删除角色 */
export const deleteRole = (id: number) => {
  return http.request<Result>("delete", `/api/v1/auth/role/${id}`);
};

/** 获取系统管理-菜单管理列表 */
export const getMenuList = (data?: object) => {
  return http.request<Result>("post", "/api/v1/auth/menu", { data });
};

/** 新增菜单 */
export const createMenu = (data?: object) => {
  return http.request<Result>("post", "/api/v1/auth/menu/create", { data });
};

/** 修改菜单 */
export const updateMenu = (id: number, data?: object) => {
  return http.request<Result>("put", `/api/v1/auth/menu/${id}`, { data });
};

/** 删除菜单 */
export const deleteMenu = (id: number) => {
  return http.request<Result>("delete", `/api/v1/auth/menu/${id}`);
};

/** 获取审计日志列表 */
export const getAuditLogs = (params?: object) => {
  return http.request<ResultTable>("get", "/api/v1/auth/audit-logs", { params });
};

/** 获取系统监控-在线用户列表 */
export const getOnlineLogsList = (data?: object) => {
  return http.request<ResultTable>("post", "/api/v1/auth/online-logs", { data });
};

/** 获取系统监控-登录日志列表 */
export const getLoginLogsList = (data?: object) => {
  return http.request<ResultTable>("post", "/api/v1/auth/login-logs", { data });
};

/** 获取系统监控-操作日志列表 */
export const getOperationLogsList = (data?: object) => {
  return http.request<ResultTable>("post", "/api/v1/auth/operation-logs", { data });
};

/** 获取系统监控-系统日志列表 */
export const getSystemLogsList = (data?: object) => {
  return http.request<ResultTable>("post", "/api/v1/auth/system-logs", { data });
};

/** 获取系统监控-系统日志-根据 id 查日志详情 */
export const getSystemLogsDetail = (data?: object) => {
  return http.request<Result>("post", "/api/v1/auth/system-logs-detail", { data });
};

/** 获取角色管理-权限-菜单权限 */
export const getRoleMenu = (data?: object) => {
  return http.request<Result>("post", "/api/v1/auth/role-menu", { data });
};

/** 获取角色管理-权限-菜单权限-根据角色 id 查对应菜单 */
export const getRoleMenuIds = (data?: object) => {
  return http.request<Result>("post", "/api/v1/auth/role-menu-ids", { data });
};

export const updateRoleMenus = (data?: object) => {
  return http.request<Result>("post", "/api/v1/auth/update-role-menus", { data });
};
