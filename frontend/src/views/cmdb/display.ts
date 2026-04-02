import type { AppItem, Cluster, PortItem } from "@/api/cmdb";
import type { CmdbListState, ResourceKey } from "./schema";

export function normalizeKeyword(value: string) {
  return value.trim().toLowerCase();
}

export function matchCluster(item: Cluster, keyword: string) {
  const hit = normalizeKeyword(keyword);
  return [item.name, item.type, item.env, item.remark].some(value =>
    String(value || "")
      .toLowerCase()
      .includes(hit)
  );
}

export function matchPort(item: PortItem, keyword: string, appMap: Map<number, AppItem>) {
  const hit = normalizeKeyword(keyword);
  const appName = appMap.get(item.app_id)?.name || "";
  return [item.port, item.protocol, item.remark, appName, item.is_public ? "public" : "private"].some(value =>
    String(value || "")
      .toLowerCase()
      .includes(hit)
  );
}

export function getColumnLabel(column: string) {
  const labels: Record<string, string> = {
    ID: "ID",
    name: "名称",
    type: "类型",
    env: "环境",
    remark: "备注",
    domain_id: "访问域名",
    address: "地址",
    cluster_id: "集群",
    status: "状态",
    host_id: "主机",
    app_id: "应用",
    version: "版本",
    deploy_type: "部署方式",
    port: "端口",
    protocol: "协议",
    is_public: "公网开放",
    domain: "域名",
    source_app_id: "源应用",
    target_app_id: "目标应用",
    source_host_id: "源主机",
    target_host_id: "目标主机",
    source_node: "源外部节点",
    target_node: "目标外部节点",
    desc: "说明",
    ip: "IP",
    public_ip: "公网IP",
    private_ip: "内网IP",
    cpu: "CPU",
    memory: "内存",
    os: "操作系统"
  };
  return labels[column] || column;
}

export function formatValue(key: ResourceKey, column: string, value: any, listState: CmdbListState) {
  if (key === "hosts" && column === "address") {
    const row = value as any;
    const segments = [
      row?.private_ip ? `内网 ${row.private_ip}` : "",
      row?.public_ip ? `公网 ${row.public_ip}` : "",
      row?.ip && row?.ip !== row?.private_ip && row?.ip !== row?.public_ip ? `兼容 ${row.ip}` : ""
    ].filter(Boolean);
    return segments.join(" / ") || "-";
  }
  if (value === null || value === undefined || value === "") return "-";
  if (typeof value === "boolean") return value ? "true" : "false";

  if (key === "hosts" && column === "cluster_id") {
    return listState.clusters.find(item => item.ID === value)?.name || value;
  }
  if (key === "apps" && column === "host_id") {
    return listState.hosts.find(item => item.ID === value)?.name || value;
  }
  if (key === "ports" && column === "app_id") {
    return listState.apps.find(item => item.ID === value)?.name || value;
  }
  if (key === "domains" && (column === "host_id" || column === "app_id")) {
    const source = column === "host_id" ? listState.hosts : listState.apps;
    return source.find(item => item.ID === value)?.name || value;
  }
  if (key === "dependencies" && column === "domain_id") {
    return listState.domains.find(item => item.ID === value)?.domain || value;
  }
  if (key === "dependencies" && column.endsWith("app_id")) {
    return listState.apps.find(item => item.ID === value)?.name || value;
  }
  if (key === "dependencies" && column.endsWith("host_id")) {
    return listState.hosts.find(item => item.ID === value)?.name || value;
  }
  return value;
}
