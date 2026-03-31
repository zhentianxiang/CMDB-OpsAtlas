# OpsAtlas 数据库常用操作命令

本文整理的是当前 `cmdb-v2` 项目里最常用的一批数据库操作命令，默认基于项目根目录执行，并以 `docker compose` + `mysql:8.0` 为前提。

## 基础连接

### 1. 查看 MySQL 容器状态

```bash
docker compose ps mysql
```

### 2. 进入 MySQL 容器

```bash
docker compose exec mysql sh
```

### 3. 使用 root 账号登录 MySQL

```bash
docker compose exec mysql mysql -uroot -prootpassword
```

### 4. 直接连接认证库 `cmdb_auth`

```bash
docker compose exec mysql mysql -uroot -prootpassword cmdb_auth
```

### 5. 直接连接资源库 `cmdb_resource`

```bash
docker compose exec mysql mysql -uroot -prootpassword cmdb_resource
```

## 库表排查

### 6. 查看所有数据库

```bash
docker compose exec mysql mysql -uroot -prootpassword -e "SHOW DATABASES;"
```

### 7. 查看认证库里的表

```bash
docker compose exec mysql mysql -uroot -prootpassword -D cmdb_auth -e "SHOW TABLES;"
```

### 8. 查看资源库里的表

```bash
docker compose exec mysql mysql -uroot -prootpassword -D cmdb_resource -e "SHOW TABLES;"
```

### 9. 查看用户表结构

```bash
docker compose exec mysql mysql -uroot -prootpassword -D cmdb_auth -e "DESC users;"
```

### 10. 查看角色表结构

```bash
docker compose exec mysql mysql -uroot -prootpassword -D cmdb_auth -e "DESC roles;"
```

## 常用查询

### 11. 查询所有用户

```bash
docker compose exec mysql mysql -uroot -prootpassword -D cmdb_auth -e "SELECT id, username, nickname, role, dept_id, status, created_at FROM users ORDER BY id DESC;"
```

### 12. 查询角色与菜单关联

```bash
docker compose exec mysql mysql -uroot -prootpassword -D cmdb_auth -e "SELECT role_id, menu_id FROM role_menus ORDER BY role_id, menu_id;"
```

### 13. 查询最近的审计日志

```bash
docker compose exec mysql mysql -uroot -prootpassword -D cmdb_resource -e "SELECT id, username, operation, path, method, status, created_at FROM audit_logs ORDER BY id DESC LIMIT 20;"
```

### 14. 查询 CMDB 主机数据

```bash
docker compose exec mysql mysql -uroot -prootpassword -D cmdb_resource -e "SELECT id, name, private_ip, public_ip, cluster_id, status FROM hosts ORDER BY id DESC;"
```

### 15. 查询 CMDB 应用数据

```bash
docker compose exec mysql mysql -uroot -prootpassword -D cmdb_resource -e "SELECT id, name, host_id, type, version, deploy_type FROM apps ORDER BY id DESC;"
```

## 备份与恢复

### 16. 手工导出认证库

```bash
docker compose exec mysql sh -c 'mysqldump -uroot -prootpassword cmdb_auth > /tmp/cmdb_auth.sql'
docker compose cp mysql:/tmp/cmdb_auth.sql ./backups/cmdb_auth-manual.sql
```

### 17. 手工导出资源库

```bash
docker compose exec mysql sh -c 'mysqldump -uroot -prootpassword cmdb_resource > /tmp/cmdb_resource.sql'
docker compose cp mysql:/tmp/cmdb_resource.sql ./backups/cmdb_resource-manual.sql
```

### 18. 从宿主机 SQL 文件恢复资源库

```bash
cat ./backups/cmdb_resource-manual.sql | docker compose exec -T mysql mysql -uroot -prootpassword cmdb_resource
```

## 运维建议

- 生产环境不要把 `rootpassword` 写死在脚本里，建议改为环境变量或单独的安全凭据文件。
- 执行 `DELETE`、`UPDATE`、`TRUNCATE` 前，先跑一遍对应的 `SELECT` 进行确认。
- 恢复 SQL 前建议先备份一次当前库，避免误覆盖。
- 如果只是临时查看数据，优先使用只读查询，不要直接在线改表。
