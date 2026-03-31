import { http } from "@/utils/http";
import { formatToken, getToken } from "@/utils/auth";

export type BaseResp<T> = {
  code: number;
  message: string;
  data: T;
};

export type ListQuery = {
  keyword?: string;
  host_id?: number;
  app_id?: number;
  source_app_id?: number;
  target_app_id?: number;
};

export type Cluster = {
  ID: number;
  name: string;
  type: string;
  env: string;
  remark: string;
};

export type Host = {
  ID: number;
  name: string;
  ip: string;
  public_ip: string;
  private_ip: string;
  cluster_id: number | null;
  cpu: number;
  memory: number;
  os: string;
  status: string;
  remark: string;
};

export type AppItem = {
  ID: number;
  name: string;
  host_id: number;
  type: string;
  version: string;
  deploy_type: string;
  remark: string;
};

export type PortItem = {
  ID: number;
  app_id: number;
  port: number;
  protocol: string;
  is_public: boolean;
  remark: string;
};

export type DomainItem = {
  ID: number;
  domain: string;
  app_id: number | null;
  host_id: number | null;
  remark: string;
};

export type DependencyItem = {
  ID: number;
  source_app_id: number | null;
  target_app_id: number | null;
  source_host_id: number | null;
  target_host_id: number | null;
  domain_id: number | null;
  source_node: string | null;
  target_node: string | null;
  desc: string;
  remark: string;
};

export type HostDetail = {
  host: {
    id: number;
    name: string;
    ip: string;
  };
  cluster?: {
    ID: number;
    name: string;
  };
  apps: Array<{
    id: number;
    name: string;
    ports: number[];
  }>;
  domains: string[];
  calls_outgoing: Array<{
    source_app_id: number | null;
    target_app_id: number | null;
    source_host_id: number | null;
    target_host_id: number | null;
    source_node: string | null;
    target_node: string | null;
    desc: string;
  }>;
  calls_incoming: Array<{
    source_app_id: number | null;
    target_app_id: number | null;
    source_host_id: number | null;
    target_host_id: number | null;
    source_node: string | null;
    target_node: string | null;
    desc: string;
  }>;
};

export type TopologyData = {
  nodes: Array<{ id: string; name: string; type: string }>;
  links: Array<{ source: string; target: string }>;
};

export type CmdbExportData = {
  version: string;
  exported_at: string;
  clusters: Cluster[];
  hosts: Host[];
  apps: AppItem[];
  ports: PortItem[];
  domains: DomainItem[];
  dependencies: DependencyItem[];
};

export type ImportMode = "overwrite" | "append";

export type ImportSummary = {
  clusters: number;
  hosts: number;
  apps: number;
  ports: number;
  domains: number;
  dependencies: number;
};

export type PreviewResourceDiff = {
  resource: string;
  add_count: number;
  skip_count: number;
  add_items: string[];
  skip_items: string[];
};

export type CmdbPreviewData = {
  append: {
    summary: ImportSummary;
    resources: PreviewResourceDiff[];
  };
  overwrite: {
    current: ImportSummary;
    incoming: ImportSummary;
  };
};

export type CmdbImportResult = {
  mode: string;
  added: ImportSummary;
  skipped?: ImportSummary;
};

export type OpsCountItem = {
  name: string;
  value: number;
};

export type OpsServiceStatus = {
  name: string;
  status: string;
  latency_ms: number;
  url: string;
  message: string;
};

export type OpsResourceTotals = {
  total_cpu: number;
  total_memory: number;
  online_hosts: number;
  offline_hosts: number;
};

export type OpsLatestApp = {
  id: number;
  name: string;
  host_name: string;
  type: string;
  deploy_type: string;
  updated_at: string;
};

export type OpsLatestDomain = {
  id: number;
  domain: string;
  app_name: string;
  host_name: string;
  updated_at: string;
};

export type OpsLatestDependency = {
  id: number;
  source: string;
  target: string;
  desc: string;
  updated_at: string;
};

export type OpsOverviewData = {
  counts: ImportSummary;
  host_status: OpsCountItem[];
  app_types: OpsCountItem[];
  deploy_types: OpsCountItem[];
  resource_totals: OpsResourceTotals;
  services: OpsServiceStatus[];
  latest_apps: OpsLatestApp[];
  latest_domains: OpsLatestDomain[];
  latest_dependencies: OpsLatestDependency[];
};

export type TransferRecordItem = {
  id: number;
  action: string;
  mode: string;
  status: string;
  filename: string;
  operator: string;
  message: string;
  detail: string;
  added: ImportSummary;
  skipped: ImportSummary;
  current: ImportSummary;
  incoming: ImportSummary;
  created_at: string;
};

export type BackupPolicy = {
  id: number;
  enabled: boolean;
  backup_hour: number;
  retention_days: number;
  backup_types: string[];
  backup_dir: string;
  last_run_at?: string;
};

export type BackupPolicyPayload = {
  enabled: boolean;
  backup_hour: number;
  retention_days: number;
  backup_types: string[];
  backup_dir: string;
};

export type BackupFileItem = {
  id: number;
  batch_no: string;
  trigger_source: string;
  backup_type: string;
  status: string;
  filename: string;
  size_bytes: number;
  message: string;
  operator: string;
  started_at: string;
  completed_at?: string;
  expires_at?: string;
};

export type SqlConsolePayload = {
  sql: string;
};

export type SqlConsoleResult = {
  statement_type: string;
  columns: string[];
  rows: Array<Record<string, any>>;
  row_count: number;
  affected_rows: number;
  elapsed_ms: number;
  truncated: boolean;
};

export type ServiceLogResult = {
  service_name: string;
  source: string;
  file_path: string;
  lines: string[];
  line_count: number;
};

function buildQuery(params?: ListQuery) {
  const query = new URLSearchParams();
  Object.entries(params || {}).forEach(([key, value]) => {
    if (value === undefined || value === null || value === "") return;
    query.set(key, String(value));
  });
  const search = query.toString();
  return search ? `?${search}` : "";
}

export const listClusters = () => http.get<BaseResp<Cluster[]>, unknown>("/api/v1/clusters");
export const listHosts = (params?: ListQuery) =>
  http.get<BaseResp<Host[]>, unknown>(`/api/v1/hosts${buildQuery(params)}`);
export const listApps = (params?: ListQuery) =>
  http.get<BaseResp<AppItem[]>, unknown>(`/api/v1/apps${buildQuery(params)}`);
export const listPorts = () => http.get<BaseResp<PortItem[]>, unknown>("/api/v1/ports");
export const listDomains = (params?: ListQuery) =>
  http.get<BaseResp<DomainItem[]>, unknown>(`/api/v1/domains${buildQuery(params)}`);
export const listDependencies = (params?: ListQuery) =>
  http.get<BaseResp<DependencyItem[]>, unknown>(`/api/v1/dependencies${buildQuery(params)}`);

export const createItem = (path: string, data: Record<string, any>) =>
  http.post<BaseResp<unknown>, unknown>(path, { data });

export const updateItem = (path: string, id: number, data: Record<string, any>) =>
  http.request<BaseResp<unknown>>("put", `${path}/${id}`, { data });

export const deleteItem = (path: string, id: number) =>
  http.request<BaseResp<unknown>>("delete", `${path}/${id}`);

export const getHostDetail = (id: number) =>
  http.get<BaseResp<HostDetail>, unknown>(`/api/v1/hosts/${id}/detail`);

export const getTopology = (clusterId?: number | null) =>
  http.get<BaseResp<TopologyData>, unknown>(
    clusterId ? `/api/v1/topology?cluster_id=${clusterId}` : "/api/v1/topology"
  );

function buildAuthHeaders() {
  const token = getToken()?.accessToken;
  return token ? { Authorization: formatToken(token) } : {};
}

export async function exportCmdbJson() {
  const response = await fetch("/api/v1/cmdb/export", {
    method: "GET",
    headers: {
      Accept: "application/json",
      ...buildAuthHeaders()
    }
  });

  if (!response.ok) {
    let errorMessage = "导出失败";
    try {
      const payload = await response.json();
      errorMessage = payload?.message || errorMessage;
    } catch {
      // ignore parse failure and use fallback message
    }
    throw new Error(errorMessage);
  }

  return response;
}

export async function previewCmdbJson(data: CmdbExportData, filename?: string) {
  const response = await fetch("/api/v1/cmdb/import/preview", {
    method: "POST",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
      ...(filename ? { "X-CMDB-Filename": filename } : {}),
      ...buildAuthHeaders()
    },
    body: JSON.stringify(data)
  });

  let payload: BaseResp<CmdbPreviewData> | null = null;

  try {
    payload = await response.json();
  } catch {
    // keep null and fall through to generic error
  }

  if (!response.ok || !payload) {
    throw new Error(payload?.message || "预览差异失败");
  }

  return payload;
}

export async function importCmdbJson(data: CmdbExportData, mode: ImportMode, filename?: string) {
  const response = await fetch(`/api/v1/cmdb/import?mode=${mode}`, {
    method: "POST",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
      ...(filename ? { "X-CMDB-Filename": filename } : {}),
      ...buildAuthHeaders()
    },
    body: JSON.stringify(data)
  });

  let payload: BaseResp<CmdbImportResult> | null = null;

  try {
    payload = await response.json();
  } catch {
    // keep null and fall through to generic error
  }

  if (!response.ok || !payload) {
    throw new Error(payload?.message || "导入失败");
  }

  return payload;
}

export const getOpsOverview = () =>
  http.get<BaseResp<OpsOverviewData>, unknown>("/api/v1/ops/overview");

export const listTransferRecords = (action?: string) =>
  http.get<BaseResp<TransferRecordItem[]>, unknown>(
    action ? `/api/v1/ops/transfer-records?action=${encodeURIComponent(action)}` : "/api/v1/ops/transfer-records"
  );

export const getServiceLogs = (params: {
  service: string;
  lines?: number;
  sinceMinutes?: number;
}) => {
  const query = new URLSearchParams();
  query.set("service", params.service);
  if (params.lines) query.set("lines", String(params.lines));
  if (params.sinceMinutes) query.set("sinceMinutes", String(params.sinceMinutes));
  return http.get<BaseResp<ServiceLogResult>, unknown>(
    `/api/v1/ops/service-logs?${query.toString()}`
  );
};

export const querySqlConsole = (data: SqlConsolePayload) =>
  http.post<BaseResp<SqlConsoleResult>, unknown>("/api/v1/ops/sql/query", {
    data
  });

export const executeSqlConsole = (data: SqlConsolePayload) =>
  http.post<BaseResp<SqlConsoleResult>, unknown>("/api/v1/ops/sql/execute", {
    data
  });

export const getBackupPolicy = () =>
  http.get<BaseResp<BackupPolicy>, unknown>("/api/v1/backup/policy");

export const updateBackupPolicy = (data: BackupPolicyPayload) =>
  http.request<BaseResp<BackupPolicy>>("put", "/api/v1/backup/policy", { data });

export const listBackupFiles = () =>
  http.get<BaseResp<BackupFileItem[]>, unknown>("/api/v1/backup/files");

export const runBackupNow = (triggerSource = "manual") =>
  http.post<BaseResp<BackupFileItem[]>, unknown>("/api/v1/backup/run", {
    data: { trigger_source: triggerSource }
  });

export const restoreBackupFile = (id: number) =>
  http.post<BaseResp<{ restored: boolean; filename: string; message: string }>, unknown>(
    `/api/v1/backup/files/${id}/restore`
  );

export async function downloadBackupFile(id: number, filename: string) {
  const response = await fetch(`/api/v1/backup/files/${id}/download`, {
    method: "GET",
    headers: {
      Accept: "application/octet-stream",
      ...buildAuthHeaders()
    }
  });

  if (!response.ok) {
    let errorMessage = "下载备份文件失败";
    try {
      const payload = await response.json();
      errorMessage = payload?.message || errorMessage;
    } catch {
      // ignore
    }
    throw new Error(errorMessage);
  }

  const blob = await response.blob();
  const url = window.URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  window.URL.revokeObjectURL(url);
}
