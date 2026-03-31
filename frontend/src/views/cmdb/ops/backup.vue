<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import dayjs from "dayjs";
import { ElMessageBox } from "element-plus";
import type {
  ImportMode,
  CmdbExportData,
  CmdbPreviewData,
  CmdbImportResult,
  TransferRecordItem,
  BackupFileItem,
  BackupPolicy
} from "@/api/cmdb";
import {
  downloadBackupFile,
  exportCmdbJson,
  getBackupPolicy,
  importCmdbJson,
  listBackupFiles,
  listTransferRecords,
  previewCmdbJson,
  restoreBackupFile,
  runBackupNow,
  updateBackupPolicy
} from "@/api/cmdb";
import { hasPerms } from "@/utils/auth";
import { message } from "@/utils/message";

defineOptions({
  name: "CMDBOpsBackup"
});

const loading = ref(false);
const importDialogVisible = ref(false);
const previewDrawerVisible = ref(false);
const importInputRef = ref<HTMLInputElement | null>(null);
const importMode = ref<ImportMode | "preview">("append");
const importFilename = ref("");
const importPayload = ref<CmdbExportData | null>(null);
const importPreview = ref<CmdbPreviewData | null>(null);
const records = ref<TransferRecordItem[]>([]);
const backupFiles = ref<BackupFileItem[]>([]);
const policy = reactive<BackupPolicy>({
  id: 0,
  enabled: false,
  backup_hour: 2,
  retention_days: 7,
  backup_types: ["json", "database"],
  backup_dir: "/app/backups",
  last_run_at: ""
});

const hourOptions = Array.from({ length: 24 }, (_, index) => ({
  label: `${String(index).padStart(2, "0")}:00`,
  value: index
}));

const recentRecords = computed(() => records.value.slice(0, 8));

async function loadBackupData() {
  try {
    loading.value = true;
    const [policyResp, filesResp, recordsResp] = await Promise.all([
      getBackupPolicy(),
      listBackupFiles(),
      listTransferRecords()
    ]);
    Object.assign(policy, policyResp.data);
    backupFiles.value = filesResp.data;
    records.value = recordsResp.data;
  } catch (error: any) {
    message(error?.message || "加载备份数据失败", { type: "error" });
  } finally {
    loading.value = false;
  }
}

async function refreshLists() {
  const [filesResp, recordsResp] = await Promise.all([listBackupFiles(), listTransferRecords()]);
  backupFiles.value = filesResp.data;
  records.value = recordsResp.data;
}

async function handleSavePolicy() {
  try {
    loading.value = true;
    const resp = await updateBackupPolicy({
      enabled: policy.enabled,
      backup_hour: policy.backup_hour,
      retention_days: policy.retention_days,
      backup_types: policy.backup_types,
      backup_dir: policy.backup_dir
    });
    Object.assign(policy, resp.data);
    message("备份策略已保存", { type: "success" });
  } catch (error: any) {
    message(error?.message || "保存备份策略失败", { type: "error" });
  } finally {
    loading.value = false;
  }
}

async function handleRunBackup() {
  try {
    loading.value = true;
    await runBackupNow("manual");
    message("手动备份执行完成", { type: "success" });
  } catch (error: any) {
    message(error?.message || "执行备份失败", { type: "error" });
  } finally {
    await refreshLists();
    await loadPolicyOnly();
    loading.value = false;
  }
}

async function loadPolicyOnly() {
  const resp = await getBackupPolicy();
  Object.assign(policy, resp.data);
}

async function handleDownloadBackup(file: BackupFileItem) {
  try {
    await downloadBackupFile(file.id, file.filename);
    message("备份文件下载成功", { type: "success" });
  } catch (error: any) {
    message(error?.message || "下载备份文件失败", { type: "error" });
  }
}

async function handleRestoreBackup(file: BackupFileItem) {
  if (file.backup_type !== "database") {
    message("只有数据库 SQL 备份支持一键恢复", { type: "warning" });
    return;
  }
  try {
    await ElMessageBox.confirm(
      `即将使用备份文件 ${file.filename} 恢复当前数据库，这会覆盖现有表数据，确认继续吗？`,
      "确认恢复备份",
      {
        confirmButtonText: "确认恢复",
        cancelButtonText: "取消",
        type: "warning",
        draggable: true,
        closeOnClickModal: false
      }
    );
  } catch {
    return;
  }

  try {
    loading.value = true;
    const resp = await restoreBackupFile(file.id);
    message(resp.data?.message || "SQL 备份恢复成功", { type: "success" });
  } catch (error: any) {
    message(error?.message || "恢复 SQL 备份失败", { type: "error" });
  } finally {
    await refreshLists();
    loading.value = false;
  }
}

async function handleExportAll() {
  try {
    loading.value = true;
    const response = await exportCmdbJson();
    const blob = await response.blob();
    const contentDisposition = response.headers.get("content-disposition") || "";
    const matched = contentDisposition.match(/filename=\"?([^\"]+)\"?/i);
    const filename = matched?.[1] || `cmdb-export-${Date.now()}.json`;
    const url = window.URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    window.URL.revokeObjectURL(url);
    await refreshLists();
    message("导出成功", { type: "success" });
  } catch (error: any) {
    message(error?.message || "导出失败", { type: "error" });
  } finally {
    loading.value = false;
  }
}

function pickImportFile() {
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

function getSummaryText(summary?: {
  clusters: number;
  hosts: number;
  apps: number;
  ports: number;
  domains: number;
  dependencies: number;
}) {
  if (!summary) return "-";
  return `集群 ${summary.clusters} / 主机 ${summary.hosts} / 应用 ${summary.apps} / 端口 ${summary.ports} / 域名 ${summary.domains} / 依赖 ${summary.dependencies}`;
}

function getResourceLabel(resource: string) {
  const labels: Record<string, string> = {
    clusters: "集群",
    hosts: "主机",
    apps: "应用",
    ports: "端口",
    domains: "域名",
    dependencies: "依赖"
  };
  return labels[resource] || resource;
}

function showImportSuccess(result?: CmdbImportResult) {
  if (!result) {
    message("导入成功", { type: "success" });
    return;
  }
  if (result.mode === "append") {
    message(`追加导入完成：新增 ${getSummaryText(result.added)}；跳过 ${getSummaryText(result.skipped)}`, {
      type: "success"
    });
    return;
  }
  message(`覆盖导入完成：${getSummaryText(result.added)}`, { type: "success" });
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
    const resp = await importCmdbJson(importPayload.value, mode, importFilename.value);
    showImportSuccess(resp.data);
    importDialogVisible.value = false;
    previewDrawerVisible.value = false;
    importPreview.value = null;
    await refreshLists();
  } catch (error: any) {
    message(error?.message || "导入失败", { type: "error" });
  } finally {
    loading.value = false;
  }
}

async function handlePreviewImport() {
  if (!importPayload.value) {
    message("请先选择 JSON 文件", { type: "warning" });
    return;
  }
  try {
    loading.value = true;
    const resp = await previewCmdbJson(importPayload.value, importFilename.value);
    importPreview.value = resp.data;
    previewDrawerVisible.value = true;
    await refreshLists();
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

function formatTime(value?: string) {
  if (!value) return "-";
  return dayjs(value).format("YYYY-MM-DD HH:mm:ss");
}

function formatFileSize(value: number) {
  if (!value) return "-";
  if (value < 1024) return `${value} B`;
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`;
  return `${(value / 1024 / 1024).toFixed(2)} MB`;
}

function getActionLabel(action: string) {
  const labels: Record<string, string> = {
    export: "导出备份",
    import: "导入恢复",
    preview: "差异预览",
    backup: "文件备份",
    restore: "备份恢复"
  };
  return labels[action] || action;
}

function getModeLabel(mode: string) {
  const labels: Record<string, string> = {
    export: "导出",
    append: "追加导入",
    overwrite: "覆盖导入",
    preview: "预览差异",
    manual: "手动执行",
    scheduled: "定时执行"
  };
  return labels[mode] || mode || "-";
}

function getBackupTypeLabel(type: string) {
  const labels: Record<string, string> = {
    json: "JSON 逻辑备份",
    database: "数据库 SQL 备份"
  };
  return labels[type] || type;
}

function getTriggerLabel(type: string) {
  const labels: Record<string, string> = {
    manual: "手动执行",
    scheduled: "定时执行"
  };
  return labels[type] || type || "-";
}

onMounted(loadBackupData);
</script>

<template>
  <div class="backup-page" v-loading="loading">
    <input ref="importInputRef" type="file" accept=".json,application/json" class="hidden-file-input" @change="handleImportFileChange" />

    <div class="backup-grid backup-grid-2">
      <el-card shadow="never">
        <template #header>备份策略</template>
        <el-form label-width="120px">
          <el-form-item label="启用定时备份">
            <el-switch v-model="policy.enabled" />
          </el-form-item>
          <el-form-item label="执行时间">
            <el-select v-model="policy.backup_hour" style="width: 220px">
              <el-option v-for="item in hourOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="保留天数">
            <el-input-number v-model="policy.retention_days" :min="1" :max="365" />
          </el-form-item>
          <el-form-item label="备份类型">
            <el-checkbox-group v-model="policy.backup_types">
              <el-checkbox label="json">JSON 逻辑备份</el-checkbox>
              <el-checkbox label="database">数据库 SQL 备份</el-checkbox>
            </el-checkbox-group>
          </el-form-item>
          <el-form-item label="备份目录">
            <el-input v-model="policy.backup_dir" placeholder="/app/backups" />
          </el-form-item>
          <el-form-item label="最近执行">
            <span class="muted-text">{{ formatTime(policy.last_run_at) }}</span>
          </el-form-item>
        </el-form>
        <div class="backup-actions">
          <el-button v-if="hasPerms('ops:backup:manage')" type="primary" @click="handleSavePolicy">保存策略</el-button>
          <el-button v-if="hasPerms('ops:backup:manage')" type="success" @click="handleRunBackup">立即执行备份</el-button>
        </div>
        <div class="backup-help">
          <p>定时备份会按策略把文件写入备份目录，建议通过 `docker-compose` 把该目录挂载到宿主机。</p>
          <p>当前支持 `JSON` 逻辑备份和 `MySQL SQL` 备份，超出保留天数的文件会自动清理。</p>
        </div>
      </el-card>

      <el-card shadow="never">
        <template #header>导出与恢复</template>
        <div class="backup-actions">
          <el-button v-if="hasPerms('cmdb:export')" type="primary" @click="handleExportAll" :loading="loading">导出全部 JSON</el-button>
          <el-button v-if="hasPerms('cmdb:import')" @click="importDialogVisible = true">导入 JSON</el-button>
        </div>
        <div class="backup-help">
          <p>`导出全部 JSON` 用于即时下载当前 CMDB 全量数据，适合迁移与人工校验。</p>
          <p>`立即执行备份` 会把备份文件落到服务器备份目录，适合保留历史与定时任务。</p>
          <p>导入支持覆盖、追加、预览差异三种模式。</p>
        </div>
      </el-card>
    </div>

    <el-card shadow="never">
      <template #header>备份文件列表</template>
      <el-table :data="backupFiles" border empty-text="暂无备份文件记录">
        <el-table-column label="执行时间" min-width="176">
          <template #default="scope">{{ formatTime(scope.row.started_at) }}</template>
        </el-table-column>
        <el-table-column label="触发方式" min-width="120">
          <template #default="scope">{{ getTriggerLabel(scope.row.trigger_source) }}</template>
        </el-table-column>
        <el-table-column label="备份类型" min-width="160">
          <template #default="scope">{{ getBackupTypeLabel(scope.row.backup_type) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="scope">
            <el-tag :type="scope.row.status === 'success' ? 'success' : scope.row.status === 'running' ? 'warning' : 'danger'">
              {{ scope.row.status === "success" ? "成功" : scope.row.status === "running" ? "执行中" : "失败" }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="filename" label="文件名" min-width="240" show-overflow-tooltip />
        <el-table-column label="文件大小" min-width="120">
          <template #default="scope">{{ formatFileSize(scope.row.size_bytes) }}</template>
        </el-table-column>
        <el-table-column label="过期时间" min-width="176">
          <template #default="scope">{{ formatTime(scope.row.expires_at) }}</template>
        </el-table-column>
        <el-table-column prop="message" label="结果摘要" min-width="220" show-overflow-tooltip />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="scope">
            <el-button link type="primary" :disabled="scope.row.status !== 'success'" @click="handleDownloadBackup(scope.row)">
              下载
            </el-button>
            <el-button
              v-if="hasPerms('ops:backup:manage')"
              link
              type="danger"
              :disabled="scope.row.status !== 'success' || scope.row.backup_type !== 'database'"
              @click="handleRestoreBackup(scope.row)"
            >
              恢复
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card shadow="never">
      <template #header>最近备份与恢复记录</template>
      <el-table :data="recentRecords" border empty-text="暂无操作记录">
        <el-table-column label="操作时间" min-width="172">
          <template #default="scope">{{ formatTime(scope.row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作类型" min-width="120">
          <template #default="scope">{{ getActionLabel(scope.row.action) }}</template>
        </el-table-column>
        <el-table-column label="执行方式" min-width="120">
          <template #default="scope">{{ getModeLabel(scope.row.mode) }}</template>
        </el-table-column>
        <el-table-column label="执行人" min-width="120">
          <template #default="scope">{{ scope.row.operator || "-" }}</template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="scope">
            <el-tag :type="scope.row.status === 'success' ? 'success' : 'danger'">
              {{ scope.row.status === "success" ? "成功" : "失败" }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="filename" label="文件名" min-width="220" show-overflow-tooltip />
        <el-table-column prop="message" label="结果摘要" min-width="220" show-overflow-tooltip />
      </el-table>
    </el-card>

    <el-dialog v-model="importDialogVisible" title="导入 JSON" width="620px">
      <el-form label-width="110px">
        <el-form-item label="导入文件">
          <div class="import-file-box">
            <div class="import-file-name">{{ importFilename || "未选择文件" }}</div>
            <el-button @click="pickImportFile">选择 JSON 文件</el-button>
          </div>
        </el-form-item>
        <el-form-item label="导入模式">
          <el-radio-group v-model="importMode">
            <el-radio label="append">追加导入</el-radio>
            <el-radio label="overwrite">覆盖导入</el-radio>
            <el-radio label="preview">预览差异</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="importDialogVisible = false">取消</el-button>
        <el-button type="primary" :disabled="!importPayload" @click="handleImportAction">
          {{ importMode === "preview" ? "开始预览" : "开始导入" }}
        </el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="previewDrawerVisible" title="导入差异预览" size="52%">
      <div class="preview-section" v-if="importPreview">
        <el-card shadow="never">
          <template #header>覆盖导入概览</template>
          <div class="preview-summary-line">当前数据：{{ getSummaryText(importPreview.overwrite.current) }}</div>
          <div class="preview-summary-line">导入文件：{{ getSummaryText(importPreview.overwrite.incoming) }}</div>
        </el-card>
        <el-card shadow="never">
          <template #header>追加导入概览</template>
          <div class="preview-summary-line">预计新增：{{ getSummaryText(importPreview.append.summary) }}</div>
          <div class="preview-resource-list">
            <div v-for="item in importPreview.append.resources" :key="item.resource" class="preview-resource-card">
              <div class="preview-resource-title">{{ getResourceLabel(item.resource) }}：新增 {{ item.add_count }}，跳过 {{ item.skip_count }}</div>
              <div v-if="item.add_items.length" class="preview-resource-block">
                <div class="preview-resource-subtitle">新增示例</div>
                <div v-for="(line, index) in item.add_items" :key="`add-${item.resource}-${index}`" class="preview-resource-item">{{ line }}</div>
              </div>
              <div v-if="item.skip_items.length" class="preview-resource-block">
                <div class="preview-resource-subtitle">跳过示例</div>
                <div v-for="(line, index) in item.skip_items" :key="`skip-${item.resource}-${index}`" class="preview-resource-item muted">{{ line }}</div>
              </div>
            </div>
          </div>
        </el-card>
      </div>
      <template #footer>
        <el-button @click="previewDrawerVisible = false">关闭</el-button>
        <el-button v-if="hasPerms('cmdb:import')" type="warning" :disabled="!importPayload" @click="runImport('append')">按追加导入</el-button>
        <el-button v-if="hasPerms('cmdb:import')" type="danger" :disabled="!importPayload" @click="runImport('overwrite')">按覆盖导入</el-button>
      </template>
    </el-drawer>
  </div>
</template>

<style scoped>
.backup-page,
.preview-section,
.preview-resource-list {
  display: grid;
  gap: 16px;
}

.backup-grid {
  display: grid;
  gap: 16px;
}

.backup-grid-2 {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.backup-actions {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.backup-help {
  margin-top: 16px;
  color: #475569;
  line-height: 1.8;
}

.muted-text {
  color: #64748b;
}

.import-file-box {
  width: 100%;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.import-file-name {
  flex: 1;
  min-width: 0;
  word-break: break-all;
  color: #334155;
}

.preview-summary-line {
  color: #334155;
  line-height: 1.8;
}

.preview-resource-card {
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  padding: 12px 14px;
}

.preview-resource-title {
  font-weight: 600;
}

.preview-resource-block {
  margin-top: 10px;
}

.preview-resource-subtitle {
  margin-bottom: 6px;
  color: #64748b;
  font-size: 12px;
}

.preview-resource-item {
  color: #334155;
  line-height: 1.6;
  word-break: break-all;
}

.preview-resource-item.muted {
  color: #64748b;
}

.hidden-file-input {
  display: none;
}

@media (max-width: 1200px) {
  .backup-grid-2 {
    grid-template-columns: 1fr;
  }
}
</style>
