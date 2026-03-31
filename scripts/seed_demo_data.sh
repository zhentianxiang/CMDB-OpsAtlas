#!/bin/bash

set -euo pipefail

ACTION="${1:-seed}"
STATE_FILE="${STATE_FILE:-scripts/.demo_seed_state.env}"
BASE_URL="${BASE_URL:-http://localhost:8090}"

AUTH_URL="${BASE_URL}/api/v1/auth"
CLUSTER_URL="${BASE_URL}/api/v1/clusters"
HOST_URL="${BASE_URL}/api/v1/hosts"
APP_URL="${BASE_URL}/api/v1/apps"
PORT_URL="${BASE_URL}/api/v1/ports"
DOMAIN_URL="${BASE_URL}/api/v1/domains"
DEPENDENCY_URL="${BASE_URL}/api/v1/dependencies"

RAND="$(date +%s)"
IP_SUFFIX=$((RAND % 200 + 20))
IP_SUFFIX_2=$(((IP_SUFFIX + 1) % 250))
IP_SUFFIX_3=$(((IP_SUFFIX + 2) % 250))
IP_SUFFIX_4=$(((IP_SUFFIX + 3) % 250))

extract_id() {
  echo "$1" | grep -Eo '"(ID|id)":[0-9]+' | head -n1 | cut -d':' -f2
}

login() {
  local login_resp
  login_resp=$(curl -s -X POST "${AUTH_URL}/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin123"}')

  TOKEN=$(echo "$login_resp" | grep -o '"token":"[^"]*' | cut -d'"' -f4)
  if [ -z "${TOKEN}" ]; then
    echo "登录失败: $login_resp"
    exit 1
  fi
}

create_entity() {
  local url="$1"
  local payload="$2"
  local resp
  resp=$(curl -s -X POST "$url" \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d "$payload")

  local id
  id=$(extract_id "$resp")
  if [ -z "$id" ]; then
    echo "创建失败 URL=$url"
    echo "响应: $resp"
    exit 1
  fi
  echo "$id"
}

delete_entity() {
  local url="$1"
  local id="$2"
  curl -s -X DELETE "${url}/${id}" -H "Authorization: Bearer ${TOKEN}" > /dev/null || true
}

seed() {
  if [ -f "$STATE_FILE" ]; then
    echo "检测到已有状态文件: $STATE_FILE"
    echo "请先执行: ./scripts/seed_demo_data.sh cleanup"
    exit 1
  fi

  login

  echo "开始创建演示数据..."

  CLUSTER_PROD_ID=$(create_entity "$CLUSTER_URL" "{\"name\":\"prod-$RAND\",\"type\":\"Kubernetes\",\"env\":\"prod\",\"remark\":\"生产环境集群\"}")
  CLUSTER_DEV_ID=$(create_entity "$CLUSTER_URL" "{\"name\":\"dev-$RAND\",\"type\":\"Kubernetes\",\"env\":\"dev\",\"remark\":\"开发环境集群\"}")

  HOST_WEB_ID=$(create_entity "$HOST_URL" "{\"name\":\"web-$RAND\",\"ip\":\"10.10.1.$IP_SUFFIX\",\"private_ip\":\"10.10.1.$IP_SUFFIX\",\"cluster_id\":$CLUSTER_PROD_ID,\"cpu\":4,\"memory\":8,\"os\":\"Ubuntu 22.04\",\"status\":\"online\",\"remark\":\"前端入口机\"}")
  HOST_API_ID=$(create_entity "$HOST_URL" "{\"name\":\"api-$RAND\",\"ip\":\"10.10.1.$IP_SUFFIX_2\",\"private_ip\":\"10.10.1.$IP_SUFFIX_2\",\"cluster_id\":$CLUSTER_PROD_ID,\"cpu\":8,\"memory\":16,\"os\":\"Ubuntu 22.04\",\"status\":\"online\",\"remark\":\"应用服务机\"}")
  HOST_DB_ID=$(create_entity "$HOST_URL" "{\"name\":\"db-$RAND\",\"ip\":\"10.10.1.$IP_SUFFIX_3\",\"private_ip\":\"10.10.1.$IP_SUFFIX_3\",\"cluster_id\":$CLUSTER_PROD_ID,\"cpu\":8,\"memory\":32,\"os\":\"Ubuntu 22.04\",\"status\":\"online\",\"remark\":\"数据库主机\"}")
  HOST_DEV_ID=$(create_entity "$HOST_URL" "{\"name\":\"devbox-$RAND\",\"ip\":\"10.10.2.$IP_SUFFIX_4\",\"private_ip\":\"10.10.2.$IP_SUFFIX_4\",\"cluster_id\":$CLUSTER_DEV_ID,\"cpu\":4,\"memory\":8,\"os\":\"Ubuntu 22.04\",\"status\":\"online\",\"remark\":\"开发测试主机\"}")

  APP_WEB_ID=$(create_entity "$APP_URL" "{\"name\":\"nginx-$RAND\",\"host_id\":$HOST_WEB_ID,\"type\":\"WEB应用\",\"version\":\"1.25\",\"deploy_type\":\"Docker\",\"remark\":\"统一流量入口\"}")
  APP_API_ID=$(create_entity "$APP_URL" "{\"name\":\"order-api-$RAND\",\"host_id\":$HOST_API_ID,\"type\":\"API服务\",\"version\":\"1.0.0\",\"deploy_type\":\"Docker\",\"remark\":\"订单服务\"}")
  APP_DB_ID=$(create_entity "$APP_URL" "{\"name\":\"mysql-$RAND\",\"host_id\":$HOST_DB_ID,\"type\":\"关系型数据库\",\"version\":\"8.0\",\"deploy_type\":\"Docker\",\"remark\":\"核心数据库\"}")
  APP_CACHE_ID=$(create_entity "$APP_URL" "{\"name\":\"redis-$RAND\",\"host_id\":$HOST_API_ID,\"type\":\"缓存\",\"version\":\"7\",\"deploy_type\":\"Docker\",\"remark\":\"缓存服务\"}")

  PORT_WEB_80_ID=$(create_entity "$PORT_URL" "{\"app_id\":$APP_WEB_ID,\"port\":80,\"protocol\":\"HTTP\",\"is_public\":true,\"remark\":\"对外访问\"}")
  PORT_WEB_443_ID=$(create_entity "$PORT_URL" "{\"app_id\":$APP_WEB_ID,\"port\":443,\"protocol\":\"HTTPS\",\"is_public\":true,\"remark\":\"HTTPS入口\"}")
  PORT_API_ID=$(create_entity "$PORT_URL" "{\"app_id\":$APP_API_ID,\"port\":8080,\"protocol\":\"HTTP\",\"is_public\":false,\"remark\":\"内网接口\"}")
  PORT_DB_ID=$(create_entity "$PORT_URL" "{\"app_id\":$APP_DB_ID,\"port\":3306,\"protocol\":\"TCP\",\"is_public\":false,\"remark\":\"数据库端口\"}")
  PORT_CACHE_ID=$(create_entity "$PORT_URL" "{\"app_id\":$APP_CACHE_ID,\"port\":6379,\"protocol\":\"TCP\",\"is_public\":false,\"remark\":\"缓存端口\"}")

  DOMAIN_MAIN_ID=$(create_entity "$DOMAIN_URL" "{\"domain\":\"api-$RAND.example.com\",\"app_id\":$APP_WEB_ID,\"host_id\":$HOST_WEB_ID,\"remark\":\"主入口域名\"}")
  DOMAIN_ADMIN_ID=$(create_entity "$DOMAIN_URL" "{\"domain\":\"admin-$RAND.example.com\",\"app_id\":$APP_WEB_ID,\"host_id\":$HOST_WEB_ID,\"remark\":\"管理后台域名\"}")

  DEP_WEB_API_ID=$(create_entity "$DEPENDENCY_URL" "{\"source_app_id\":$APP_WEB_ID,\"target_app_id\":$APP_API_ID,\"source_host_id\":$HOST_WEB_ID,\"target_host_id\":$HOST_API_ID,\"desc\":\"web calls api\",\"remark\":\"入口转发\"}")
  DEP_API_DB_ID=$(create_entity "$DEPENDENCY_URL" "{\"source_app_id\":$APP_API_ID,\"target_app_id\":$APP_DB_ID,\"source_host_id\":$HOST_API_ID,\"target_host_id\":$HOST_DB_ID,\"desc\":\"api query db\",\"remark\":\"数据库访问\"}")
  DEP_API_CACHE_ID=$(create_entity "$DEPENDENCY_URL" "{\"source_app_id\":$APP_API_ID,\"target_app_id\":$APP_CACHE_ID,\"source_host_id\":$HOST_API_ID,\"target_host_id\":$HOST_API_ID,\"desc\":\"api use cache\",\"remark\":\"缓存访问\"}")

  cat > "$STATE_FILE" <<EOS
CLUSTER_PROD_ID=$CLUSTER_PROD_ID
CLUSTER_DEV_ID=$CLUSTER_DEV_ID
HOST_WEB_ID=$HOST_WEB_ID
HOST_API_ID=$HOST_API_ID
HOST_DB_ID=$HOST_DB_ID
HOST_DEV_ID=$HOST_DEV_ID
APP_WEB_ID=$APP_WEB_ID
APP_API_ID=$APP_API_ID
APP_DB_ID=$APP_DB_ID
APP_CACHE_ID=$APP_CACHE_ID
PORT_WEB_80_ID=$PORT_WEB_80_ID
PORT_WEB_443_ID=$PORT_WEB_443_ID
PORT_API_ID=$PORT_API_ID
PORT_DB_ID=$PORT_DB_ID
PORT_CACHE_ID=$PORT_CACHE_ID
DOMAIN_MAIN_ID=$DOMAIN_MAIN_ID
DOMAIN_ADMIN_ID=$DOMAIN_ADMIN_ID
DEP_WEB_API_ID=$DEP_WEB_API_ID
DEP_API_DB_ID=$DEP_API_DB_ID
DEP_API_CACHE_ID=$DEP_API_CACHE_ID
EOS

  echo "演示数据创建完成。"
  echo "状态文件: $STATE_FILE"
}

cleanup() {
  if [ ! -f "$STATE_FILE" ]; then
    echo "未找到状态文件: $STATE_FILE"
    echo "无需清理，或先执行 seed。"
    exit 0
  fi

  # shellcheck disable=SC1090
  source "$STATE_FILE"
  login

  echo "开始清理演示数据..."

  delete_entity "$DEPENDENCY_URL" "$DEP_WEB_API_ID"
  delete_entity "$DEPENDENCY_URL" "$DEP_API_DB_ID"
  delete_entity "$DEPENDENCY_URL" "$DEP_API_CACHE_ID"

  delete_entity "$DOMAIN_URL" "$DOMAIN_MAIN_ID"
  delete_entity "$DOMAIN_URL" "$DOMAIN_ADMIN_ID"

  delete_entity "$PORT_URL" "$PORT_WEB_80_ID"
  delete_entity "$PORT_URL" "$PORT_WEB_443_ID"
  delete_entity "$PORT_URL" "$PORT_API_ID"
  delete_entity "$PORT_URL" "$PORT_DB_ID"
  delete_entity "$PORT_URL" "$PORT_CACHE_ID"

  delete_entity "$APP_URL" "$APP_WEB_ID"
  delete_entity "$APP_URL" "$APP_API_ID"
  delete_entity "$APP_URL" "$APP_DB_ID"
  delete_entity "$APP_URL" "$APP_CACHE_ID"

  delete_entity "$HOST_URL" "$HOST_WEB_ID"
  delete_entity "$HOST_URL" "$HOST_API_ID"
  delete_entity "$HOST_URL" "$HOST_DB_ID"
  delete_entity "$HOST_URL" "$HOST_DEV_ID"

  delete_entity "$CLUSTER_URL" "$CLUSTER_PROD_ID"
  delete_entity "$CLUSTER_URL" "$CLUSTER_DEV_ID"

  rm -f "$STATE_FILE"
  echo "清理完成。"
}

case "$ACTION" in
  seed)
    seed
    ;;
  cleanup)
    cleanup
    ;;
  *)
    echo "用法: ./scripts/seed_demo_data.sh [seed|cleanup]"
    exit 1
    ;;
esac
