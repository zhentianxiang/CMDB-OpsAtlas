# CMDB Frontend (vue-pure-admin)

该目录已切换为 `vue-pure-admin` 模板，并接入 CMDB 微服务。

## 登录

- 用户名: `admin`
- 密码: `admin123`

## API

前端统一请求 `/api/v1/*`，由 Nginx 转发到对应后端服务。

## 本地构建（Docker）

在项目根目录执行：

```bash
docker compose up --build
```

访问地址：`http://localhost:8090`
