<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { ElMessageBox } from "element-plus";
import { message } from "@/utils/message";
import { hasPerms } from "@/utils/auth";
import G6Topology from "./components/G6Topology.vue";
import CmdbImportPanel from "./components/CmdbImportPanel.vue";
import HostDetailPanel from "./components/HostDetailPanel.vue";
import CmdbResourceTable from "./components/CmdbResourceTable.vue";
import CmdbTopologyToolbar from "./components/CmdbTopologyToolbar.vue";
import { matchCluster, matchPort } from "./display";
import {
  configs,
  resourcePermissionMap,
  searchPlaceholders,
  type CmdbTab,
  type PortDraft,
  type ResourceKey
} from "./schema";
import {
  createItem,
  deleteItem,
  importCmdbJson,
  previewCmdbJson,
  listApps,
  listClusters,
  listDependencies,
  listDomains,
  listHosts,
  listPorts,
  updateItem,
  getHostDetail,
  type AppItem,
  type CmdbImportResult,
  type CmdbExportData,
  type CmdbPreviewData,
  type Cluster,
  type DependencyItem,
  type DomainItem,
  type Host,
  type HostDetail,
  type ImportMode,
  type PortItem
} from "@/api/cmdb";

type TopologyClickPayload =
  | {
      nodeType: "host";
      host: Host;
      clusterName: string;
      appsText: string;
      portsText: string;
    }
  | {
      nodeType: "app";
      app: AppItem;
      hostName: string;
      ports: string[];
      domains: string[];
      inCount: number;
      outCount: number;
      laneName: string;
    }
  | {
      nodeType: "external";
      name: string;
      laneName: string;
      sourceHint: string;
    };

const props = withDefaults(
  defineProps<{
    defaultTab?: CmdbTab;
  }>(),
  {
    defaultTab: "clusters"
  }
);

const route = useRoute();
const activeTab = ref<CmdbTab>(props.defaultTab);
const currentPage = ref(1);
const pageSize = ref(20);
const currentPageHostApps = ref(1);
const currentPageHostOutgoing = ref(1);
const currentPageHostIncoming = ref(1);
const loading = ref(false);
const dialogVisible = ref(false);
const editingId = ref<number | null>(null);
const editingKey = ref<ResourceKey>("clusters");
const tableRef = ref<{ clearSelection: () => void } | null>(null);
const importInputRef = ref<HTMLInputElement | null>(null);
const importDialogVisible = ref(false);
const previewDrawerVisible = ref(false);
const importMode = ref<ImportMode | "preview">("append");
const importFilename = ref("");
const importPayload = ref<CmdbExportData | null>(null);
const importPreview = ref<CmdbPreviewData | null>(null);

const nodeDetailVisible = ref(false);
const nodeDetailTitle = ref("");
const nodeDetailRows = ref<Array<{ label: string; value: string }>>([]);

const formModel = reactive<Record<string, any>>({});
const hostDetail = ref<HostDetail | null>(null);
const selectedHostId = ref<number | null>(null);
const selectedClusterId = ref<number | null>(null);
const selectedTopologyHostId = ref<number | null>(null);
const selectedTopologyAppId = ref<number | null>(null);
const selectedTopologyDomainId = ref<number | null>(null);
const searchTimer = ref<number | null>(null);

const listState = reactive<{
  clusters: Cluster[];
  hosts: Host[];
  apps: AppItem[];
  ports: PortItem[];
  domains: DomainItem[];
  dependencies: DependencyItem[];
}>({
  clusters: [],
  hosts: [],
  apps: [],
  ports: [],
  domains: [],
  dependencies: []
});

const searchState = reactive<Record<ResourceKey, string>>({
  clusters: "",
  hosts: "",
  apps: "",
  ports: "",
  domains: "",
  dependencies: ""
});

const selectionState = reactive<Record<ResourceKey, number[]>>({
  clusters: [],
  hosts: [],
  apps: [],
  ports: [],
  domains: [],
  dependencies: []
});

const searchResultState = reactive<{
  hosts: Host[] | null;
  apps: AppItem[] | null;
  domains: DomainItem[] | null;
  dependencies: DependencyItem[] | null;
}>({
  hosts: null,
  apps: null,
  domains: null,
  dependencies: null
});

const activeConfig = computed(
  () => configs.find(item => item.key === activeTab.value) || configs[0]
);


const allActiveRows = computed<any[]>(() => {
  if (["host-detail", "topology"].includes(activeTab.value)) return [];
  const key = activeTab.value as ResourceKey;
  const keyword = searchState[key]?.trim();
  if (!keyword) return listState[key] || [];

  if (key === "clusters") {
    return listState.clusters.filter(item => matchCluster(item, keyword));
  }
  if (key === "ports") {
    return listState.ports.filter(item => matchPort(item, keyword, appMap.value));
  }
  if (key === "hosts" || key === "apps" || key === "domains" || key === "dependencies") {
    return searchResultState[key] || [];
  }
  return listState[key] || [];
});

const activeRows = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value;
  return allActiveRows.value.slice(start, start + pageSize.value);
});

const hostMap = computed(() => {
  const map = new Map<number, Host>();
  listState.hosts.forEach(item => map.set(item.ID, item));
  return map;
});

const paginatedHostApps = computed(() => {
  if (!hostDetail.value) return [];
  const start = (currentPageHostApps.value - 1) * pageSize.value;
  return (hostDetail.value.apps || []).slice(start, start + pageSize.value);
});

const paginatedHostOutgoing = computed(() => {
  if (!hostDetail.value) return [];
  const start = (currentPageHostOutgoing.value - 1) * pageSize.value;
  return (hostDetail.value.calls_outgoing || []).slice(start, start + pageSize.value);
});

const paginatedHostIncoming = computed(() => {
  if (!hostDetail.value) return [];
  const start = (currentPageHostIncoming.value - 1) * pageSize.value;
  return (hostDetail.value.calls_incoming || []).slice(start, start + pageSize.value);
});

const appMap = computed(() => {
  const map = new Map<number, AppItem>();
  listState.apps.forEach(item => map.set(item.ID, item));
  return map;
});

const topologyHostOptions = computed(() =>
  listState.hosts.filter(item => {
    if (!selectedClusterId.value) return true;
    return item.cluster_id === selectedClusterId.value;
  })
);

const topologyAppOptions = computed(() => {
  return listState.apps.filter(item => {
    // 如果选择了具体主机，则只显示该主机的应用 (层级递归)
    if (selectedTopologyHostId.value) {
      return item.host_id === selectedTopologyHostId.value;
    }
    // 如果只选择了集群，则显示该集群下所有主机的应用
    if (selectedClusterId.value) {
      const host = listState.hosts.find(h => h.ID === item.host_id);
      return host?.cluster_id === selectedClusterId.value;
    }
    // 否则显示全部
    return true;
  });
});

const topologyDomainOptions = computed(() => {
  // 域名筛选不遵循层级规则，显示全部以供全局检索
  return listState.domains;
});

const searchPlaceholder = computed(() => {
  return searchPlaceholders[activeTab.value as ResourceKey] || "搜索";
});

const searchVisible = computed(() => activeTab.value !== "host-detail" && activeTab.value !== "topology");

const currentSearchKeyword = computed({
  get: () => {
    if (activeTab.value === "host-detail" || activeTab.value === "topology") return "";
    return searchState[activeTab.value as ResourceKey];
  },
  set: value => {
    if (activeTab.value === "host-detail" || activeTab.value === "topology") return;
    searchState[activeTab.value as ResourceKey] = value;
  }
});

const activeSelectionIds = computed<number[]>(() => {
  if (activeTab.value === "host-detail" || activeTab.value === "topology") return [];
  return selectionState[activeTab.value as ResourceKey] || [];
});

const canBatchDelete = computed(
  () => activeTab.value !== "host-detail" && activeTab.value !== "topology" && activeSelectionIds.value.length > 0
);

const canCreateCurrent = computed(() => {
  if (activeTab.value === "host-detail" || activeTab.value === "topology") return false;
  return hasPerms(resourcePermissionMap[activeTab.value as ResourceKey].create);
});

const canUpdateCurrent = computed(() => {
  if (activeTab.value === "host-detail" || activeTab.value === "topology") return false;
  return hasPerms(resourcePermissionMap[activeTab.value as ResourceKey].update);
});

const canDeleteCurrent = computed(() => {
  if (activeTab.value === "host-detail" || activeTab.value === "topology") return false;
  return hasPerms(resourcePermissionMap[activeTab.value as ResourceKey].delete);
});

const protocolOptions = ["TCP", "UDP", "HTTP", "HTTPS"];
const portDrafts = ref<PortDraft[]>([]);
const isPortCreateMode = computed(() => editingKey.value === "ports" && !editingId.value);

function getSelectOptions(type?: "clusters" | "hosts" | "apps") {
  if (!type) return [];
  return listState[type];
}

function getDependencyAppOptions(side: "source" | "target") {
  const hostKey = side === "source" ? "source_host_id" : "target_host_id";
  const selectedHostId = Number(formModel[hostKey]);
  if (!selectedHostId) return [];
  return listState.apps.filter(item => item.host_id === selectedHostId);
}

function getFieldSelectOptions(field: {
  key: string;
  optionsFrom?: "clusters" | "hosts" | "apps";
  options?: Array<{ label: string; value: string | number }>;
}) {
  if (field.options) return field.options;
  if (editingKey.value === "dependencies" && field.key === "source_app_id") {
    return getDependencyAppOptions("source");
  }
  if (editingKey.value === "dependencies" && field.key === "target_app_id") {
    return getDependencyAppOptions("target");
  }
  if (editingKey.value === "dependencies" && field.key === "domain_id") {
    const sHostId = Number(formModel.source_host_id);
    const sAppId = Number(formModel.source_app_id);
    if (!sHostId && !sAppId) return listState.domains;
    return listState.domains.filter(item => {
      if (sAppId) return item.app_id === sAppId;
      if (sHostId) return item.host_id === sHostId || (item.app_id && appMap.value.get(item.app_id)?.host_id === sHostId);
      return true;
    });
  }
  if ((editingKey.value === "ports" || editingKey.value === "domains") && field.key === "app_id") {
    const selectedHostId = Number(formModel.host_id);
    if (!selectedHostId) return getSelectOptions(field.optionsFrom);
    return listState.apps.filter(item => item.host_id === selectedHostId);
  }
  return getSelectOptions(field.optionsFrom);
}

function formatAppOption(item: AppItem) {
  return item.name;
}

function getSelectOptionLabel(
  field: { key: string; optionsFrom?: "clusters" | "hosts" | "apps" },
  item: any
) {
  if (field.key === "domain_id") {
    return item.domain;
  }
  if ("ID" in item && "name" in item) {
    if ((field.optionsFrom === "apps" || field.key.includes("app_id")) && "host_id" in item) {
      return formatAppOption(item as AppItem);
    }
    return item.name;
  }
  return item.label;
}

function isFieldSelectDisabled(field: { key: string }) {
  return false;
}

function getFieldSelectPlaceholder(field: { key: string; label: string }) {
  if (editingKey.value === "dependencies" && field.key === "source_app_id") {
    return formModel.source_host_id ? "选择调用方应用" : "可先选调用方主机，或直接选应用";
  }
  if (editingKey.value === "dependencies" && field.key === "target_app_id") {
    return formModel.target_host_id ? "选择被调用应用" : "可先选被调用方主机，或直接选应用";
  }
  if (editingKey.value === "ports" && field.key === "app_id") {
    return formModel.host_id ? "选择应用" : "可先选主机，或直接选应用";
  }
  return `请选择${field.label}`;
}

function syncDependencyHostFromApp(side: "source" | "target") {
  const appKey = side === "source" ? "source_app_id" : "target_app_id";
  const hostKey = side === "source" ? "source_host_id" : "target_host_id";
  const appId = Number(formModel[appKey]);
  if (!appId) return;
  const app = listState.apps.find(item => item.ID === appId);
  if (app) {
    formModel[hostKey] = app.host_id;
  }
}

function syncPortHostFromApp() {
  const appId = Number(formModel.app_id);
  if (!appId) return;
  const app = listState.apps.find(item => item.ID === appId);
  if (app) {
    formModel.host_id = app.host_id;
  }
}

function validateDependencyPayload(payload: Record<string, any>) {
  const pairs: Array<{
    hostKey: "source_host_id" | "target_host_id";
    appKey: "source_app_id" | "target_app_id";
    hostLabel: string;
    appLabel: string;
  }> = [
    {
      hostKey: "source_host_id",
      appKey: "source_app_id",
      hostLabel: "调用方主机",
      appLabel: "调用方应用"
    },
    {
      hostKey: "target_host_id",
      appKey: "target_app_id",
      hostLabel: "被调用主机",
      appLabel: "被调用应用"
    }
  ];

  for (const pair of pairs) {
    const appId = payload[pair.appKey];
    const hostId = payload[pair.hostKey];
    if (!appId && !hostId) continue;

    if (appId && !hostId) {
      const app = listState.apps.find(item => item.ID === Number(appId));
      if (!app) {
        throw new Error(`${pair.appLabel}不存在`);
      }
      payload[pair.hostKey] = app.host_id;
      continue;
    }

    if (appId && hostId) {
      const app = listState.apps.find(item => item.ID === Number(appId));
      if (!app) {
        throw new Error(`${pair.appLabel}不存在`);
      }
      if (app.host_id !== Number(hostId)) {
        throw new Error(`${pair.hostLabel}和${pair.appLabel}不匹配`);
      }
    }
  }
}

function createEmptyPortDraft(): PortDraft {
  return {
    port: null,
    protocol: "TCP"
  };
}

function resetPortDrafts(row: any = null) {
  if (editingKey.value !== "ports") {
    portDrafts.value = [];
    return;
  }

  if (!row) {
    portDrafts.value = [createEmptyPortDraft()];
    return;
  }

  portDrafts.value = [
    {
      port: row.port ?? null,
      protocol: row.protocol || "TCP"
    }
  ];
}

function addPortDraft() {
  portDrafts.value.push(createEmptyPortDraft());
}

function removePortDraft(index: number) {
  if (portDrafts.value.length === 1) {
    portDrafts.value[0] = createEmptyPortDraft();
    return;
  }
  portDrafts.value.splice(index, 1);
}


function openCreate() {
  editingId.value = null;
  editingKey.value = activeConfig.value.key;
  formModelReset();
  dialogVisible.value = true;
}

function openEdit(row: any) {
  editingId.value = row.ID;
  editingKey.value = activeConfig.value.key;
  formModelReset(row);
  dialogVisible.value = true;
}

function formModelReset(row: any = null) {
  const conf = activeConfig.value;
  conf.fields.forEach(field => {
    if (!row) {
      if (field.type === "checkbox") formModel[field.key] = false;
      else if (field.key === "protocol") formModel[field.key] = "TCP";
      else formModel[field.key] = "";
      return;
    }
    if (field.type === "checkbox") formModel[field.key] = Boolean(row[field.key]);
    else formModel[field.key] = row[field.key] ?? "";
  });

  if (editingKey.value === "dependencies") {
    if (!formModel.source_host_id && formModel.source_app_id) {
      const sourceApp = listState.apps.find(item => item.ID === Number(formModel.source_app_id));
      if (sourceApp) formModel.source_host_id = sourceApp.host_id;
    }
    if (!formModel.target_host_id && formModel.target_app_id) {
      const targetApp = listState.apps.find(item => item.ID === Number(formModel.target_app_id));
      if (targetApp) formModel.target_host_id = targetApp.host_id;
    }
  }

  if (editingKey.value === "ports") {
    if (!formModel.host_id && formModel.app_id) {
      const app = listState.apps.find(item => item.ID === Number(formModel.app_id));
      if (app) formModel.host_id = app.host_id;
    }
  }

  resetPortDrafts(row);
}

async function handleSave() {
  const conf = configs.find(item => item.key === editingKey.value)!;
  const payload: Record<string, any> = {};

  for (const field of conf.fields) {
    const raw = formModel[field.key];
    if (field.type === "checkbox") {
      payload[field.key] = Boolean(raw);
      continue;
    }
    if (field.type === "number") {
      payload[field.key] = raw === "" || raw === null ? 0 : Number(raw);
      continue;
    }
    if (field.type === "select") {
      if (raw === "" || raw === null || raw === undefined) {
        payload[field.key] = field.nullable ? null : 0;
      } else if (field.options) {
        payload[field.key] = raw;
      } else {
        payload[field.key] = Number(raw);
      }
      continue;
    }
    payload[field.key] = typeof raw === "string" ? raw.trim() : raw;
  }

  if (editingKey.value === "dependencies") {
    validateDependencyPayload(payload);
  }

  if (editingKey.value === "ports") {
    delete payload.host_id;
  }

  try {
    loading.value = true;
    if (editingKey.value === "ports" && !editingId.value) {
      const appId = Number(payload.app_id);
      const remark = typeof payload.remark === "string" ? payload.remark.trim() : payload.remark;
      const isPublic = Boolean(payload.is_public);
      const validDrafts = portDrafts.value.filter(
        item => item.port !== null && item.port !== undefined && String(item.port).trim() !== ""
      );

      if (!appId) {
        throw new Error("请选择应用");
      }
      if (validDrafts.length === 0) {
        throw new Error("请至少添加一个端口");
      }

      await Promise.all(
        validDrafts.map(item =>
          createItem(conf.path, {
            app_id: appId,
            port: Number(item.port),
            protocol: item.protocol || "TCP",
            is_public: isPublic,
            remark
          })
        )
      );
    } else if (editingId.value) {
      await updateItem(conf.path, editingId.value, payload);
    } else {
      await createItem(conf.path, payload);
    }
    message("保存成功", { type: "success" });
    dialogVisible.value = false;
    await loadAll();
  } catch (error: any) {
    message(error?.message || "保存失败", { type: "error" });
  } finally {
    loading.value = false;
  }
}

async function handleDelete(row: any) {
  const conf = activeConfig.value;
  try {
    loading.value = true;
    await deleteItem(conf.path, row.ID);
    message("删除成功", { type: "success" });
    await loadAll();
  } catch (error: any) {
    message(error?.message || "删除失败", { type: "error" });
  } finally {
    loading.value = false;
  }
}

function handleSelectionChange(rows: any[]) {
  if (activeTab.value === "host-detail" || activeTab.value === "topology") return;
  selectionState[activeTab.value as ResourceKey] = rows.map(item => Number(item.ID));
}

function clearSelectionForKey(key: ResourceKey) {
  selectionState[key] = [];
  tableRef.value?.clearSelection();
}

async function handleBatchDelete() {
  if (activeTab.value === "host-detail" || activeTab.value === "topology") return;

  const key = activeTab.value as ResourceKey;
  const ids = [...selectionState[key]];
  if (ids.length === 0) return;

  const conf = activeConfig.value;
  try {
    loading.value = true;
    const results = await Promise.allSettled(ids.map(id => deleteItem(conf.path, id)));
    const successCount = results.filter(item => item.status === "fulfilled").length;
    const failedCount = results.length - successCount;

    if (successCount > 0) {
      message(
        failedCount > 0
          ? `批量删除完成，成功 ${successCount} 条，失败 ${failedCount} 条`
          : `批量删除成功，共删除 ${successCount} 条`,
        { type: failedCount > 0 ? "warning" : "success" }
      );
    } else {
      message("批量删除失败", { type: "error" });
    }

    clearSelectionForKey(key);
    await loadAll();
  } catch (error: any) {
    message(error?.message || "批量删除失败", { type: "error" });
  } finally {
    loading.value = false;
  }
}

function pickImportFile() {
  if (loading.value) return;
  importInputRef.value?.click();
}

async function handleImportFileChange(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = "";
  if (!file) return;

  try {
    const text = await file.text();
    importPayload.value = JSON.parse(text) as CmdbExportData;
    importFilename.value = file.name;
    importPreview.value = null;
    message(`已选择 ${file.name}`, { type: "success" });
  } catch (error: any) {
    importPayload.value = null;
    importFilename.value = "";
    importPreview.value = null;
    message(error?.message || "JSON 文件解析失败", { type: "error" });
  }
}

async function runImport(mode: ImportMode) {
  if (!importPayload.value) {
    message("请先选择 JSON 文件", { type: "warning" });
    return;
  }

  try {
    await ElMessageBox.confirm(
      mode === "overwrite"
        ? "覆盖导入会清空当前所有 CMDB 数据，此操作不可恢复，确认继续吗？"
        : "追加导入会保留现有数据，仅导入新增项，确认继续吗？",
      mode === "overwrite" ? "确认覆盖导入" : "确认追加导入",
      {
        confirmButtonText: "继续导入",
        cancelButtonText: "取消",
        type: mode === "overwrite" ? "warning" : "info",
        draggable: true,
        closeOnClickModal: false
      }
    );
  } catch {
    return;
  }

  try {
    loading.value = true;
    const resp = await importCmdbJson(importPayload.value, mode);
    showImportSuccess(resp.data);
    importDialogVisible.value = false;
    previewDrawerVisible.value = false;
    importPreview.value = null;
    await loadAll();
  } catch (error: any) {
    message(error?.message || "导入失败", { type: "error" });
  } finally {
    loading.value = false;
  }
}

function showImportSuccess(result?: CmdbImportResult) {
  if (!result) {
    message("导入成功", { type: "success" });
    return;
  }

  if (result.mode === "append") {
    message(
      `追加导入完成：新增 ${getImportSummaryText(result.added)}；跳过 ${getImportSummaryText(result.skipped)}`,
      { type: "success" }
    );
    return;
  }

  message(`覆盖导入完成：${getImportSummaryText(result.added)}`, { type: "success" });
}

async function handlePreviewImport() {
  if (!importPayload.value) {
    message("请先选择 JSON 文件", { type: "warning" });
    return;
  }

  try {
    loading.value = true;
    const resp = await previewCmdbJson(importPayload.value);
    importPreview.value = resp.data;
    previewDrawerVisible.value = true;
  } catch (error: any) {
    message(error?.message || "预览差异失败", { type: "error" });
  } finally {
    loading.value = false;
  }
}

async function handleImportAction() {
  if (importMode.value === "preview") {
    await handlePreviewImport();
    return;
  }
  await runImport(importMode.value);
}

async function runSearchForKey(key: ResourceKey) {
  const keyword = searchState[key]?.trim();
  if (key !== "hosts" && key !== "apps" && key !== "domains" && key !== "dependencies") return;

  if (!keyword) {
    searchResultState[key] = null;
    return;
  }

  try {
    loading.value = true;
    if (key === "hosts") {
      const resp = await listHosts({ keyword });
      searchResultState.hosts = resp.data || [];
      return;
    }
    if (key === "apps") {
      const resp = await listApps({ keyword });
      searchResultState.apps = resp.data || [];
      return;
    }
    if (key === "domains") {
      const resp = await listDomains({ keyword });
      searchResultState.domains = resp.data || [];
      return;
    }
    if (key === "dependencies") {
      const resp = await listDependencies({ keyword });
      searchResultState.dependencies = resp.data || [];
    }
  } catch (error: any) {
    message(error?.message || "搜索失败", { type: "error" });
  } finally {
    loading.value = false;
  }
}

function scheduleSearch() {
  if (searchTimer.value) window.clearTimeout(searchTimer.value);
  const key = activeTab.value;
  if (key === "host-detail" || key === "topology") return;

  searchTimer.value = window.setTimeout(() => {
    runSearchForKey(key as ResourceKey);
  }, 250);
}

function clearCurrentSearch() {
  if (activeTab.value === "host-detail" || activeTab.value === "topology") return;
  currentSearchKeyword.value = "";
  scheduleSearch();
}

async function loadAll() {
  try {
    loading.value = true;
    const [c, h, a, p, d, dep] = await Promise.all([
      listClusters(),
      listHosts(),
      listApps(),
      listPorts(),
      listDomains(),
      listDependencies()
    ]);
    listState.clusters = c.data || [];
    listState.hosts = h.data || [];
    listState.apps = a.data || [];
    listState.ports = p.data || [];
    listState.domains = d.data || [];
    listState.dependencies = dep.data || [];
    Object.keys(selectionState).forEach(key => {
      selectionState[key as ResourceKey] = [];
    });
    searchResultState.hosts = null;
    searchResultState.apps = null;
    searchResultState.domains = null;
    searchResultState.dependencies = null;
    if (!selectedHostId.value && listState.hosts.length > 0) {
      selectedHostId.value = listState.hosts[0].ID;
    }
    if (
      activeTab.value !== "host-detail" &&
      activeTab.value !== "topology" &&
      searchState[activeTab.value as ResourceKey].trim()
    ) {
      await runSearchForKey(activeTab.value as ResourceKey);
    }

  } catch (error: any) {
    message(error?.message || "加载数据失败", { type: "error" });
  } finally {
    loading.value = false;
  }
}

async function queryHostDetail() {
  if (!selectedHostId.value) return;
  try {
    loading.value = true;
    const resp = await getHostDetail(selectedHostId.value);
    hostDetail.value = resp.data;
  } catch (error: any) {
    message(error?.message || "查询主机详情失败", { type: "error" });
  } finally {
    loading.value = false;
  }
}

function handleTopologyNodeClick(payload: TopologyClickPayload) {
  if (payload.nodeType === "host") {
    const host = payload.host;
    nodeDetailTitle.value = `主机节点 - ${host.name}`;
    nodeDetailRows.value = [
      { label: "主机ID", value: String(host.ID) },
      { label: "IP", value: host.private_ip || host.public_ip || host.ip || "-" },
      { label: "状态", value: host.status || "-" },
      { label: "集群", value: payload.clusterName || "-" },
      { label: "应用", value: payload.appsText || "-" },
      { label: "端口明细", value: payload.portsText || "-" }
    ];
  } else if (payload.nodeType === "app") {
    const app = payload.app;
    nodeDetailTitle.value = `应用节点 - ${app.name}`;
    nodeDetailRows.value = [
      { label: "应用ID", value: String(app.ID) },
      { label: "部署主机", value: payload.hostName || "-" },
      { label: "层级", value: payload.laneName || "-" },
      { label: "类型", value: app.type || "-" },
      { label: "版本", value: app.version || "-" },
      { label: "部署方式", value: app.deploy_type || "-" },
      { label: "端口", value: (payload.ports || []).join("\n") || "-" },
      { label: "域名", value: (payload.domains || []).join("\n") || "-" },
      { label: "调用关系", value: `出向 ${payload.outCount || 0} / 入向 ${payload.inCount || 0}` }
    ];
  } else {
    nodeDetailTitle.value = `外部节点 - ${payload.name}`;
    nodeDetailRows.value = [
      { label: "节点类型", value: "外部/未登记节点" },
      { label: "名称", value: payload.name || "-" },
      { label: "层级", value: payload.laneName || "-" },
      { label: "说明", value: payload.sourceHint || "-" }
    ];
  }
  nodeDetailVisible.value = true;
}

function formatDependencyEndpoint(item: DependencyItem, side: "source" | "target") {
  const appId = side === "source" ? item.source_app_id : item.target_app_id;
  const hostId = side === "source" ? item.source_host_id : item.target_host_id;
  const nodeText = side === "source" ? item.source_node : item.target_node;
  if (appId) return appMap.value.get(appId)?.name || "-";
  if (hostId) return hostMap.value.get(hostId)?.name || "-";
  if (nodeText && nodeText.trim()) return nodeText.trim();
  return "-";
}

async function queryTopology() {
  await loadAll();
}

function resetTopologyFilters() {
  selectedTopologyHostId.value = null;
  selectedTopologyAppId.value = null;
  selectedTopologyDomainId.value = null;
}

function syncTabFromRoute() {
  if (props.defaultTab) {
    activeTab.value = props.defaultTab;
    return;
  }
  if (route.path.endsWith("/topology")) activeTab.value = "topology";
  else activeTab.value = "clusters";
}

watch(
  () => route.fullPath,
  () => {
    syncTabFromRoute();
  }
);

onMounted(async () => {
  syncTabFromRoute();
  await loadAll();
  if (selectedHostId.value) await queryHostDetail();
});

watch(currentSearchKeyword, () => {
  scheduleSearch();
});

watch(
  () => activeTab.value,
  value => {
    if (value === "host-detail" || value === "topology") return;
    clearSelectionForKey(value as ResourceKey);
    if (searchState[value as ResourceKey].trim()) {
      scheduleSearch();
    }
  }
);

watch(selectedClusterId, () => {
  selectedTopologyHostId.value = null;
  selectedTopologyAppId.value = null;
  selectedTopologyDomainId.value = null;
});

watch(selectedTopologyHostId, value => {
  if (value == null) {
    selectedTopologyAppId.value = null;
    selectedTopologyDomainId.value = null;
    return;
  }
  const hostAppIds = new Set(listState.apps.filter(item => item.host_id === value).map(item => item.ID));
  if (selectedTopologyAppId.value && !hostAppIds.has(selectedTopologyAppId.value)) {
    selectedTopologyAppId.value = null;
  }
  const keepDomain = listState.domains.some(item => {
    if (selectedTopologyDomainId.value == null || getEntityIdForDomain(item) !== selectedTopologyDomainId.value) return false;
    return item.host_id === value || (item.app_id != null && hostAppIds.has(item.app_id));
  });
  if (!keepDomain) selectedTopologyDomainId.value = null;
});

watch(selectedTopologyAppId, value => {
  if (value == null) {
    selectedTopologyDomainId.value = null;
    return;
  }
  const app = listState.apps.find(item => item.ID === value);
  if (app) {
    selectedTopologyHostId.value = app.host_id;
  }
  const keepDomain = listState.domains.some(item => {
    if (selectedTopologyDomainId.value == null || getEntityIdForDomain(item) !== selectedTopologyDomainId.value) return false;
    return item.app_id === value;
  });
  if (!keepDomain) selectedTopologyDomainId.value = null;
});

watch(selectedTopologyDomainId, value => {
  if (value == null) return;
  const domain = listState.domains.find(item => getEntityIdForDomain(item) === value);
  if (!domain) return;
  if (domain.app_id) {
    selectedTopologyAppId.value = domain.app_id;
    const app = listState.apps.find(item => item.ID === domain.app_id);
    if (app) selectedTopologyHostId.value = app.host_id;
    return;
  }
  if (domain.host_id) {
    selectedTopologyHostId.value = domain.host_id;
  }
});

watch(
  () => formModel.source_host_id,
  value => {
    if (editingKey.value !== "dependencies") return;
    if (!value) {
      formModel.source_app_id = "";
      return;
    }
    const exists = listState.apps.some(
      item => item.ID === Number(formModel.source_app_id) && item.host_id === Number(value)
    );
    if (!exists) {
      formModel.source_app_id = "";
    }
  }
);

watch(
  () => formModel.source_app_id,
  () => {
    if (editingKey.value !== "dependencies") return;
    syncDependencyHostFromApp("source");
  }
);

watch(
  () => formModel.target_host_id,
  value => {
    if (editingKey.value !== "dependencies") return;
    if (!value) {
      formModel.target_app_id = "";
      return;
    }
    const exists = listState.apps.some(
      item => item.ID === Number(formModel.target_app_id) && item.host_id === Number(value)
    );
    if (!exists) {
      formModel.target_app_id = "";
    }
  }
);

watch(
  () => formModel.target_app_id,
  () => {
    if (editingKey.value !== "dependencies") return;
    syncDependencyHostFromApp("target");
  }
);

watch(
  () => formModel.domain_id,
  value => {
    if (editingKey.value !== "dependencies" || !value) return;
    const dom = listState.domains.find(item => item.ID === Number(value));
    if (dom) {
      if (dom.app_id) {
        formModel.source_app_id = dom.app_id;
        const app = listState.apps.find(a => a.ID === dom.app_id);
        if (app) formModel.source_host_id = app.host_id;
      } else if (dom.host_id) {
        formModel.source_host_id = dom.host_id;
      }
    }
  }
);

watch(
  () => formModel.host_id,
  value => {
    if (editingKey.value !== "ports") return;
    if (!value) {
      formModel.app_id = "";
      return;
    }
    const exists = listState.apps.some(
      item => item.ID === Number(formModel.app_id) && item.host_id === Number(value)
    );
    if (!exists) {
      formModel.app_id = "";
    }
  }
);

watch(
  () => formModel.app_id,
  () => {
    if (editingKey.value !== "ports") return;
    syncPortHostFromApp();
  }
);

onBeforeUnmount(() => {
  if (searchTimer.value) window.clearTimeout(searchTimer.value);
});

function getEntityIdForDomain(item: DomainItem) {
  return Number(item.ID);
}
</script>

<template>
  <div class="cmdb-page">
    <el-card shadow="never">
      <template #header>
        <div class="flex justify-between items-center">
          <el-tabs v-model="activeTab">
            <el-tab-pane label="集群" name="clusters" />
            <el-tab-pane label="主机" name="hosts" />
            <el-tab-pane label="应用" name="apps" />
            <el-tab-pane label="端口" name="ports" />
            <el-tab-pane label="域名" name="domains" />
            <el-tab-pane label="依赖" name="dependencies" />
            <el-tab-pane label="主机详情" name="host-detail" />
            <el-tab-pane label="拓扑图" name="topology" />
          </el-tabs>
          <el-space>
            <el-input
              v-if="searchVisible"
              v-model="currentSearchKeyword"
              :placeholder="searchPlaceholder"
              clearable
              style="width: 320px"
              @clear="clearCurrentSearch"
            />
            <input ref="importInputRef" type="file" accept=".json,application/json" class="hidden-file-input" @change="handleImportFileChange" />
            <el-button @click="loadAll">刷新</el-button>
            <el-popconfirm
              v-if="activeTab !== 'host-detail' && activeTab !== 'topology' && canDeleteCurrent"
              :title="`确认删除已选中的 ${activeSelectionIds.length} 条${activeConfig.title}数据？`"
              @confirm="handleBatchDelete"
            >
              <template #reference>
                <el-button
                  v-if="activeTab !== 'host-detail' && activeTab !== 'topology' && canDeleteCurrent"
                  type="danger"
                  plain
                  :disabled="!canBatchDelete"
                >
                  一键删除
                  <template v-if="activeSelectionIds.length"> ({{ activeSelectionIds.length }}) </template>
                </el-button>
              </template>
            </el-popconfirm>
            <el-button
              v-if="activeTab !== 'host-detail' && activeTab !== 'topology' && canCreateCurrent"
              type="primary"
              @click="openCreate"
            >
              新增{{ activeConfig.title }}
            </el-button>
          </el-space>
        </div>
      </template>

      <div v-loading="loading">
        <template v-if="activeTab !== 'host-detail' && activeTab !== 'topology'">
          <CmdbResourceTable
            ref="tableRef"
            :active-config="activeConfig"
            :current-search-keyword="currentSearchKeyword"
            :all-active-rows="allActiveRows"
            :active-rows="activeRows"
            :list-state="listState"
            :can-update-current="canUpdateCurrent"
            :can-delete-current="canDeleteCurrent"
            :current-page="currentPage"
            :page-size="pageSize"
            @selection-change="handleSelectionChange"
            @edit="openEdit"
            @delete="handleDelete"
            @update:current-page="currentPage = $event"
            @update:page-size="pageSize = $event"
          />
        </template>

        <template v-if="activeTab === 'host-detail'">
          <HostDetailPanel
            :hosts="listState.hosts"
            :selected-host-id="selectedHostId"
            :host-detail="hostDetail"
            :page-size="pageSize"
            :current-page-host-apps="currentPageHostApps"
            :current-page-host-outgoing="currentPageHostOutgoing"
            :current-page-host-incoming="currentPageHostIncoming"
            :paginated-host-apps="paginatedHostApps"
            :paginated-host-outgoing="paginatedHostOutgoing"
            :paginated-host-incoming="paginatedHostIncoming"
            :format-dependency-endpoint="formatDependencyEndpoint"
            @update:selected-host-id="selectedHostId = $event"
            @query="queryHostDetail"
            @update:current-page-host-apps="currentPageHostApps = $event"
            @update:current-page-host-outgoing="currentPageHostOutgoing = $event"
            @update:current-page-host-incoming="currentPageHostIncoming = $event"
          />
        </template>

        <template v-if="activeTab === 'topology'">
          <CmdbTopologyToolbar
            :clusters="listState.clusters"
            :topology-host-options="topologyHostOptions"
            :topology-app-options="topologyAppOptions"
            :topology-domain-options="topologyDomainOptions"
            :selected-cluster-id="selectedClusterId"
            :selected-topology-host-id="selectedTopologyHostId"
            :selected-topology-app-id="selectedTopologyAppId"
            :selected-topology-domain-id="selectedTopologyDomainId"
            @update:selected-cluster-id="selectedClusterId = $event"
            @update:selected-topology-host-id="selectedTopologyHostId = $event"
            @update:selected-topology-app-id="selectedTopologyAppId = $event"
            @update:selected-topology-domain-id="selectedTopologyDomainId = $event"
            @query="queryTopology"
            @reset="resetTopologyFilters"
          />

          <G6Topology
            :hosts="listState.hosts"
            :apps="listState.apps"
            :dependencies="listState.dependencies"
            :ports="listState.ports"
            :domains="listState.domains"
            :clusters="listState.clusters"
            :selected-cluster-id="selectedClusterId"
            :selected-host-id="selectedTopologyHostId"
            :selected-app-id="selectedTopologyAppId"
            :selected-domain-id="selectedTopologyDomainId"
            @node-click="handleTopologyNodeClick"
          />
        </template>
      </div>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="`${editingId ? '编辑' : '新增'}${activeConfig.title}`" width="620px">
      <el-form label-width="110px">
        <el-form-item v-for="field in activeConfig.fields" :key="field.key" :label="field.label">
          <el-switch v-if="field.type === 'checkbox'" v-model="formModel[field.key]" />
          <template v-else-if="isPortCreateMode && field.key === 'port'">
            <div class="port-draft-list">
              <div v-for="(item, index) in portDrafts" :key="index" class="port-draft-row">
                <el-input-number v-model="item.port" :min="1" :max="65535" style="width: 100%" />
                <el-select filterable v-model="item.protocol" style="width: 160px">
                  <el-option v-for="protocol in protocolOptions" :key="protocol" :label="protocol" :value="protocol" />
                </el-select>
                <el-button link type="danger" @click="removePortDraft(index)">删除</el-button>
              </div>
              <el-button type="primary" link @click="addPortDraft">+ 新增一条端口</el-button>
            </div>
          </template>
          <template v-else-if="isPortCreateMode && field.key === 'protocol'">
            <span class="form-tip">每条端口可单独选择协议</span>
          </template>
          <el-select
            v-else-if="field.type === 'select'"
            v-model="formModel[field.key]"
            :clearable="!field.options"
            :disabled="isFieldSelectDisabled(field)"
            :placeholder="getFieldSelectPlaceholder(field)"
            style="width: 100%"
          >
            <el-option
              v-for="item in getFieldSelectOptions(field)"
              :key="'ID' in item ? item.ID : item.value"
              :label="getSelectOptionLabel(field, item)"
              :value="'ID' in item ? item.ID : item.value"
            />
          </el-select>
          <el-input-number v-else-if="field.type === 'number'" v-model="formModel[field.key]" style="width: 100%" />
          <el-input v-else v-model="formModel[field.key]" clearable />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSave">保存</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="nodeDetailVisible" :title="nodeDetailTitle" size="38%">
      <el-descriptions :column="1" border>
        <el-descriptions-item v-for="item in nodeDetailRows" :key="item.label" :label="item.label">
          <pre class="node-detail-text">{{ item.value }}</pre>
        </el-descriptions-item>
      </el-descriptions>
    </el-drawer>

    <CmdbImportPanel
      :dialog-visible="importDialogVisible"
      :preview-drawer-visible="previewDrawerVisible"
      :import-filename="importFilename"
      :import-payload="importPayload"
      :import-preview="importPreview"
      :import-mode="importMode"
      @update:dialog-visible="importDialogVisible = $event"
      @update:preview-drawer-visible="previewDrawerVisible = $event"
      @update:import-mode="importMode = $event"
      @pick-file="pickImportFile"
      @submit="handleImportAction"
      @run-import="runImport"
    />
  </div>
</template>

<style scoped>
.cmdb-page {
  display: grid;
  gap: 12px;
}

.node-detail-text {
  white-space: pre-wrap;
  margin: 0;
  font-family: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace;
}

.table-hint {
  margin-bottom: 12px;
  color: #5b6472;
  font-size: 13px;
}

.port-draft-list {
  width: 100%;
  display: grid;
  gap: 10px;
}

.port-draft-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 160px 56px;
  gap: 10px;
  align-items: center;
}

.form-tip {
  color: #5b6472;
  font-size: 13px;
}

.hidden-file-input {
  display: none;
}
</style>
