import { http } from "@/utils/http";
import { getToken } from "@/utils/auth";

export type UserResult = {
  code: number;
  message: string;
  data: {
    /** 头像 */
    avatar: string;
    /** 用户名 */
    username: string;
    /** 昵称 */
    nickname: string;
    /** 当前登录用户的角色 */
    roles: Array<string>;
    /** 按钮级别权限 */
    permissions: Array<string>;
    /** `token` */
    accessToken: string;
    /** 用于调用刷新`accessToken`的接口时所需的`token` */
    refreshToken: string;
    /** `accessToken`的过期时间（格式'xxxx/xx/xx xx:xx:xx'） */
    expires: Date;
  };
};

export type RefreshTokenResult = {
  code: number;
  message: string;
  data: {
    /** `token` */
    accessToken: string;
    /** 用于调用刷新`accessToken`的接口时所需的`token` */
    refreshToken: string;
    /** `accessToken`的过期时间（格式'xxxx/xx/xx xx:xx:xx'） */
    expires: Date;
  };
};

export type UserInfo = {
  /** ID */
  id?: number;
  /** 头像 */
  avatar: string;
  /** 用户名 */
  username: string;
  /** 昵称 */
  nickname: string;
  /** 邮箱 */
  email: string;
  /** 联系电话 */
  phone: string;
  /** 简介 */
  description: string;
  /** 角色 */
  role?: string;
  /** 性别 0: 男, 1: 女 */
  sex?: number;
  /** 部门 ID */
  deptId?: number;
  /** 部门详情 */
  dept?: any;
};

export type UserInfoResult = {
  code: number;
  message: string;
  data: UserInfo;
};

export type UpdateMinePayload = Pick<
  UserInfo,
  "nickname" | "email" | "phone" | "avatar" | "description" | "sex" | "deptId"
>;

export type ChangePasswordPayload = {
  oldPassword: string;
  newPassword: string;
  confirmPassword: string;
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

type AvatarUploadResult = {
  code: number;
  message: string;
  data?: {
    avatar: string;
    user?: UserInfo;
  };
};

/** 登录 */
export const getLogin = (data?: object) => {
  return http
    .request<{
      code: number;
      message: string;
      data: {
        token: string;
        user: {
          id?: number;
          username: string;
          nickname?: string;
          avatar?: string;
          email?: string;
          phone?: string;
          description?: string;
          role?: string;
        };
        permissions?: string[];
      };
    }>("post", "/api/v1/auth/login", { data })
    .then(resp => {
      const expires = new Date(Date.now() + 24 * 60 * 60 * 1000);
      return {
        code: resp.code,
        message: resp.message,
        data: {
          avatar: resp.data?.user?.avatar || "",
          username: resp.data?.user?.username || "",
          nickname:
            resp.data?.user?.nickname || resp.data?.user?.username || "",
          roles: [resp.data?.user?.role || "admin"],
          permissions: resp.data?.permissions || [],
          accessToken: resp.data?.token || "",
          refreshToken: resp.data?.token || "",
          expires
        }
      } as UserResult;
    });
};

/** 刷新`token` */
export const refreshTokenApi = (data?: object) => {
  const token = getToken();
  return Promise.resolve({
    code: 0,
    message: "success",
    data: {
      accessToken: token?.accessToken || "",
      refreshToken: token?.refreshToken || "",
      expires: new Date(Date.now() + 24 * 60 * 60 * 1000)
    }
  } as RefreshTokenResult);
};

/** 账户设置-个人信息 */
export const getMine = () =>
  http
    .request<{
      code: number;
      message: string;
      data: UserInfo;
    }>("get", "/api/v1/auth/me")
    .then(resp => ({
      code: resp.code,
      message: resp.message,
      data: resp.data
    }) as UserInfoResult);

export const updateMine = (data: UpdateMinePayload) =>
  http
    .request<{
      code: number;
      message: string;
      data: UserInfo;
    }>("put", "/api/v1/auth/me", { data })
    .then(resp => ({
      code: resp.code,
      message: resp.message,
      data: resp.data
    }) as UserInfoResult);

export const updateMyPassword = (data: ChangePasswordPayload) =>
  http
    .request<{ code: number; message: string; data?: { message?: string } }>(
      "put",
      "/api/v1/auth/password",
      { data }
    )
    .then(resp => ({
      code: resp.code,
      message: resp.message,
      data: resp.data
    }));

export const uploadMyAvatar = (file: Blob) => {
  const formData = new FormData();
  formData.append("file", file, "avatar.png");

  return http.request<AvatarUploadResult>("post", "/api/v1/auth/me/avatar", {
    data: formData
  });
};

/** 账户设置-个人安全日志 */
export const getMineLogs = (data?: object) => {
  return Promise.resolve({
    code: 0,
    message: "success",
    data: {
      list: []
    }
  } as ResultTable);
};
