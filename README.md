# CMDB v2

一个基于 `Go + Gin + GORM + MySQL + Vue 3 + vue-pure-admin + Docker Compose` 的轻量 CMDB 与运维管理平台。

当前版本已经不是单纯的“资产展示页”，而是演进成了一套包含 `CMDB 管理 + 运维管理 + 系统管理 + RBAC 权限控制` 的完整后台。

## 1. 当前能力

- `CMDB 管理`
  - 集群、主机、应用、端口、域名、依赖统一管理
  - 主机详情页
  - 拓扑图展示
- `运维管理`
  - 运行状态
  - 日志中心
  - 任务中心
  - 备份恢复
  - SQL 控制台
- `系统管理`
  - 用户管理
  - 角色管理
  - 菜单管理
  - 部门管理
  - 审计日志
- `RBAC`
  - 用户绑定角色
  - 角色绑定菜单和权限点
  - 前端按钮级控制
  - 后端接口级控制
- `运维增强`
  - 文件化服务日志
  - JSON 导入导出
  - 导入差异预览
  - 覆盖导入 / 追加导入
  - 自动备份策略
  - MySQL SQL 备份与恢复
  - SQL 查询与受控执行
- `用户能力`
  - 用户头像裁剪上传
  - 个人信息维护
  - 修改密码

## 2. 项目结构

```text
cmdb-v2/
├── README.md
├── docker-compose.yml
├── docs/
│   └── database-commands.md
├── pkg/
│   ├── common/                # 统一响应
│   ├── db/                    # 数据库连接
│   ├── logging/               # 文件日志初始化
│   ├── middleware/            # JWT、RBAC、审计中间件
│   ├── models/                # CMDB 公共 GORM 模型
│   └── utils/
├── services/
│   ├── auth-service/          # 认证、用户、角色、菜单、部门、审计、头像
│   ├── cluster-service/       # 集群服务
│   ├── host-service/          # 主机服务
│   ├── app-service/           # 应用服务
│   ├── port-service/          # 端口服务
│   ├── domain-service/        # 域名服务
│   ├── dependency-service/    # 依赖服务
│   ├── topology-service/      # 拓扑聚合服务
│   └── cmdb-transfer-service/ # 导入导出、备份恢复、日志、SQL、任务
├── frontend/                  # Vue 3 前端管理台
├── logs/                      # 服务日志宿主机挂载目录
├── backups/                   # 备份文件宿主机挂载目录
└── uploads/                   # 上传文件宿主机挂载目录
```

## 3. 架构说明

### 3.1 微服务划分

| 服务 | 端口 | 职责 |
| --- | --- | --- |
| `auth-service` | `8081` | 登录、当前用户、密码修改、用户/角色/菜单/部门、审计、头像 |
| `cluster-service` | `8082` | 集群 CRUD |
| `host-service` | `8083` | 主机 CRUD、主机详情 |
| `app-service` | `8084` | 应用 CRUD |
| `port-service` | `8085` | 端口 CRUD |
| `domain-service` | `8086` | 域名 CRUD |
| `dependency-service` | `8087` | 依赖关系 CRUD |
| `topology-service` | `8088` | 拓扑聚合查询 |
| `cmdb-transfer-service` | `8089` | JSON 导入导出、备份恢复、日志、SQL、任务 |
| `frontend` | `8090` | Web UI + Nginx API 反向代理 |
| `mysql` | `3306` | 数据库 |

### 3.2 前端代理

浏览器统一访问：

- 前端地址：`http://localhost:8090`
- API 前缀：`http://localhost:8090/api/v1/...`

`frontend/nginx/default.conf` 会按路由把请求转发到对应后端服务，所以前端和调试脚本通常只需要访问 `8090`。

### 3.3 数据库划分

项目当前使用两个数据库：

- `cmdb_auth`
  - 用户、角色、菜单、部门、角色菜单等认证与系统管理数据
- `cmdb_resource`
  - CMDB 资源、依赖、拓扑、导入导出记录、备份策略、备份文件、任务记录、审计日志等

## 4. 菜单与权限设计

### 4.1 CMDB 管理

- 集群
- 主机
- 应用
- 端口
- 域名
- 依赖
- 主机详情
- 拓扑图

### 4.2 运维管理

- 运行状态
- 日志中心
- 任务中心
- 备份恢复
- SQL 控制台

### 4.3 系统管理

- 用户管理
- 角色管理
- 菜单管理
- 部门管理
- 审计日志

### 4.4 RBAC 说明

- 用户不直接绑定零散权限，而是先绑定角色。
- 角色通过菜单和按钮权限点获得能力。
- 前端按 `permissions` 控制菜单、按钮、操作入口显示。
- 后端按 RBAC 中间件统一控制接口权限。
- `admin` 拥有最高权限。
- 普通只读用户可以查看 CMDB，但不会看到新建、编辑、删除、导入、导出、备份恢复、执行 SQL 等高风险操作。

## 5. 主要功能说明

### 5.1 CMDB 资源页

- 集群、主机、应用、端口、域名、依赖统一管理
- 支持批量删除
- 应用类型、部署方式下拉选择
- 端口支持一次录入多个端口
- 依赖关系支持主机与应用联动
- 主机详情与拓扑联动展示

### 5.2 运维管理

- `运行状态`
  - 查看服务健康状态与基础运行信息
- `日志中心`
  - 读取各服务文件日志
  - 支持按服务筛选日志来源
- `任务中心`
  - 查看导入、导出、备份、SQL 执行等记录
- `备份恢复`
  - 手动立即备份
  - 自动备份策略
  - JSON 备份
  - SQL 备份
  - SQL 一键恢复
- `SQL 控制台`
  - 只读查询模式
  - 有权限用户可切换执行模式
  - 二次确认
  - 执行历史回放

### 5.3 系统管理

- 用户新增、编辑、删除、重置密码
- 用户角色分配
- 菜单权限配置
- 部门树管理
- 角色菜单绑定
- 审计日志查询

### 5.4 个人设置与头像

- 支持修改个人资料
- 支持修改密码
- 支持头像裁剪上传
- 用户管理页和个人设置页都可上传头像
- 头像文件持久化保存到 `uploads/avatars`

## 6. 日志、备份、上传目录

这些目录都通过 `docker-compose.yml` 挂载到宿主机，容器重启或重建后文件仍会保留：

- `logs/`
  - 服务运行日志
- `backups/`
  - JSON 备份、SQL 备份文件
- `uploads/avatars/`
  - 用户头像文件

说明：

- 日志中心读取的是 `logs/*.log`
- 备份恢复页操作的是 `backups/*`
- 用户头像访问路径由 `auth-service` 静态暴露为 `/api/v1/auth/avatars/...`

## 7. 快速启动

### 7.1 环境要求

- Docker
- Docker Compose

### 7.2 启动

在项目根目录执行：

```bash
docker compose up -d --build
```

启动后访问：

- 前端：`http://localhost:8090`
- MySQL：`localhost:3306`

### 7.3 默认账号

认证服务首次启动时会自动创建默认管理员：

- 用户名：`admin`
- 密码：`admin123`

同时会自动初始化默认部门、默认角色、默认菜单和 RBAC 绑定关系。

## 8. 演示数据

项目提供了演示数据脚本：

- [scripts/seed_demo_data.sh](./scripts/seed_demo_data.sh)

创建演示数据：

```bash
./scripts/seed_demo_data.sh seed
```

清理演示数据：

```bash
./scripts/seed_demo_data.sh cleanup
```

## 9. 接口与认证规范

### 9.1 统一返回结构

所有成功接口统一返回：

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

错误时通常返回：

```json
{
  "code": 400,
  "message": "错误描述"
}
```

实现见：

- [pkg/common/response.go](./pkg/common/response.go)

### 9.2 认证方式

除登录接口外，其余接口均要求携带 JWT：

```http
Authorization: Bearer <token>
```

JWT 中会带上当前用户角色和权限集合，后端会基于权限点进行 RBAC 校验。

## 10. 常用开发与排障命令

查看服务状态：

```bash
docker compose ps
```

查看前端日志：

```bash
docker compose logs -f frontend
```

查看认证服务日志：

```bash
docker compose logs -f auth-service
```

执行后端测试：

```bash
go test ./...
```

## 11. 数据库操作文档

数据库连接、查询、备份、恢复常用命令见：

- [docs/database-commands.md](./docs/database-commands.md)

## 12. 说明

- `frontend/README.md` 仍主要保留 `vue-pure-admin` 模板原始说明。
- 本 README 以当前 `cmdb-v2` 项目实际落地状态为准，更适合作为仓库根入口文档。
