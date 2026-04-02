<script setup lang="ts">
import type { CmdbImportResult, CmdbExportData, CmdbPreviewData, ImportMode } from "@/api/cmdb";

const props = defineProps<{
  dialogVisible: boolean;
  previewDrawerVisible: boolean;
  importFilename: string;
  importPayload: CmdbExportData | null;
  importPreview: CmdbPreviewData | null;
  importMode: ImportMode | "preview";
}>();

const emit = defineEmits<{
  (e: "update:dialogVisible", value: boolean): void;
  (e: "update:previewDrawerVisible", value: boolean): void;
  (e: "update:importMode", value: ImportMode | "preview"): void;
  (e: "pick-file"): void;
  (e: "submit"): void;
  (e: "run-import", value: ImportMode): void;
}>();

function getImportSummaryText(summary?: CmdbImportResult["added"]) {
  if (!summary) return "-";
  return `集群 ${summary.clusters} / 主机 ${summary.hosts} / 应用 ${summary.apps} / 端口 ${summary.ports} / 域名 ${summary.domains} / 依赖 ${summary.dependencies}`;
}

function getPreviewResourceLabel(resource: string) {
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
</script>

<template>
  <el-dialog
    :model-value="dialogVisible"
    title="导入 JSON"
    width="620px"
    @update:model-value="emit('update:dialogVisible', $event)"
  >
    <el-form label-width="110px">
      <el-form-item label="导入文件">
        <div class="import-file-box">
          <div class="import-file-name">{{ importFilename || "未选择文件" }}</div>
          <el-button @click="emit('pick-file')">选择 JSON 文件</el-button>
        </div>
      </el-form-item>
      <el-form-item label="导入模式">
        <el-radio-group :model-value="importMode" @update:model-value="emit('update:importMode', $event)">
          <el-radio label="append">追加导入</el-radio>
          <el-radio label="overwrite">覆盖导入</el-radio>
          <el-radio label="preview">预览差异</el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="说明">
        <div class="import-help">
          <div>追加导入：保留现有数据，只导入新内容，重复项会自动跳过。</div>
          <div>覆盖导入：清空当前 CMDB 数据，再按 JSON 文件完整恢复。</div>
          <div>预览差异：先分析 JSON 和当前库之间的差异，再决定是否执行覆盖或追加。</div>
        </div>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="emit('update:dialogVisible', false)">取消</el-button>
      <el-button type="primary" :disabled="!importPayload" @click="emit('submit')">
        {{ importMode === "preview" ? "开始预览" : "开始导入" }}
      </el-button>
    </template>
  </el-dialog>

  <el-drawer
    :model-value="previewDrawerVisible"
    title="导入差异预览"
    size="52%"
    @update:model-value="emit('update:previewDrawerVisible', $event)"
  >
    <div class="preview-section" v-if="importPreview">
      <el-card shadow="never">
        <template #header>覆盖导入概览</template>
        <div class="preview-summary-line">当前数据：{{ getImportSummaryText(importPreview.overwrite.current) }}</div>
        <div class="preview-summary-line">导入文件：{{ getImportSummaryText(importPreview.overwrite.incoming) }}</div>
      </el-card>

      <el-card shadow="never">
        <template #header>追加导入概览</template>
        <div class="preview-summary-line">预计新增：{{ getImportSummaryText(importPreview.append.summary) }}</div>
        <div class="preview-resource-list">
          <div v-for="item in importPreview.append.resources" :key="item.resource" class="preview-resource-card">
            <div class="preview-resource-title">
              {{ getPreviewResourceLabel(item.resource) }}：新增 {{ item.add_count }}，跳过 {{ item.skip_count }}
            </div>
            <div v-if="item.add_items.length" class="preview-resource-block">
              <div class="preview-resource-subtitle">新增示例</div>
              <div v-for="(line, index) in item.add_items" :key="`add-${item.resource}-${index}`" class="preview-resource-item">
                {{ line }}
              </div>
            </div>
            <div v-if="item.skip_items.length" class="preview-resource-block">
              <div class="preview-resource-subtitle">跳过示例</div>
              <div
                v-for="(line, index) in item.skip_items"
                :key="`skip-${item.resource}-${index}`"
                class="preview-resource-item muted"
              >
                {{ line }}
              </div>
            </div>
          </div>
        </div>
      </el-card>
    </div>
    <template #footer>
      <el-button @click="emit('update:previewDrawerVisible', false)">关闭</el-button>
      <el-button type="warning" :disabled="!importPayload" @click="emit('run-import', 'append')">按追加导入</el-button>
      <el-button type="danger" :disabled="!importPayload" @click="emit('run-import', 'overwrite')">按覆盖导入</el-button>
    </template>
  </el-drawer>
</template>

<style scoped>
.import-file-box {
  width: 100%;
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
}

.import-file-name {
  flex: 1;
  min-width: 0;
  color: #334155;
  word-break: break-all;
}

.import-help {
  display: grid;
  gap: 8px;
  color: #5b6472;
  font-size: 13px;
  line-height: 1.6;
}

.preview-section {
  display: grid;
  gap: 16px;
}

.preview-summary-line {
  color: #334155;
  line-height: 1.8;
}

.preview-resource-list {
  display: grid;
  gap: 12px;
}

.preview-resource-card {
  border: 1px solid #dbe5f0;
  border-radius: 12px;
  padding: 12px 14px;
  background: #fbfdff;
}

.preview-resource-title {
  font-weight: 600;
  color: #1f2937;
}

.preview-resource-block {
  margin-top: 10px;
}

.preview-resource-subtitle {
  font-size: 12px;
  color: #64748b;
  margin-bottom: 6px;
}

.preview-resource-item {
  font-size: 13px;
  color: #334155;
  line-height: 1.6;
  word-break: break-all;
}

.preview-resource-item.muted {
  color: #64748b;
}
</style>
