<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import dayjs from "dayjs";
import {
  getServiceLogs,
  listTransferRecords,
  type ServiceLogResult,
  type TransferRecordItem
} from "@/api/cmdb";
import { getAuditLogs } from "@/api/system";
import { message } from "@/utils/message";

defineOptions({
  name: "OpsLogs"
});

const loading = ref(false);
const auditLogs = ref<any[]>([]);
const transferLogs = ref<TransferRecordItem[]>([]);
const serviceLog = ref<ServiceLogResult | null>(null);
const serviceName = ref("cmdb-transfer-service");
const lines = ref(200);
const sinceMinutes = ref(30);

const serviceOptions = [
  "auth-service",
  "cluster-service",
  "host-service",
  "app-service",
  "port-service",
  "domain-service",
  "dependency-service",
  "topology-service",
  "cmdb-transfer-service"
];

const logSummary = computed(() => [
  { label: "操作审计", value: auditLogs.value.length },
  { label: "运维记录", value: transferLogs.value.length },
  { label: "服务日志", value: serviceLog.value?.line_count || 0 }
]);

async function loadData() {
  try {
    loading.value = true;
    const [auditResp, transferResp] = await Promise.all([
      getAuditLogs(),
      listTransferRecords()
    ]);
    auditLogs.value = auditResp.data?.list || [];
    transferLogs.value = transferResp.data || [];
    await loadServiceLogs();
  } catch (error: any) {
    message(error?.message || "加载日志中心失败", { type: "error" });
  } finally {
    loading.value = false;
  }
}

async function loadServiceLogs() {
  try {
    const resp = await getServiceLogs({
      service: serviceName.value,
      lines: lines.value,
      sinceMinutes: sinceMinutes.value
    });
    serviceLog.value = resp.data;
  } catch (error: any) {
    serviceLog.value = null;
    message(error?.message || "获取服务日志失败", { type: "error" });
  }
}

function formatTime(value?: string) {
  return value ? dayjs(value).format("YYYY-MM-DD HH:mm:ss") : "-";
}

onMounted(loadData);
</script>

<template>
  <div class="logs-page" v-loading="loading">
    <div class="summary-grid">
      <el-card v-for="item in logSummary" :key="item.label" shadow="hover">
        <div class="summary-label">{{ item.label }}</div>
        <div class="summary-value">{{ item.value }}</div>
      </el-card>
    </div>

    <el-card shadow="never">
      <template #header>服务日志文件</template>
      <div class="toolbar">
        <el-select v-model="serviceName" style="width: 220px">
          <el-option v-for="item in serviceOptions" :key="item" :label="item" :value="item" />
        </el-select>
        <el-input-number v-model="lines" :min="20" :max="1000" />
        <el-input-number v-model="sinceMinutes" :min="1" :max="1440" />
        <el-button type="primary" @click="loadServiceLogs">刷新日志</el-button>
      </div>
      <div class="tips">
        <p>当前实现会读取各服务写入共享 `logs` 目录的日志文件，不再依赖 Docker Socket。</p>
        <p>服务启动后会同时输出到控制台和 `/app/logs/&lt;service&gt;.log`，前端展示读取到的最近日志。</p>
      </div>
      <div v-if="serviceLog" class="service-log-panel">
        <div class="service-log-meta">
          <span>服务：{{ serviceLog.service_name }}</span>
          <span>来源：{{ serviceLog.source }}</span>
          <span>文件：{{ serviceLog.file_path }}</span>
          <span>行数：{{ serviceLog.line_count }}</span>
        </div>
        <pre class="service-log-content">{{ serviceLog.lines.join("\n") }}</pre>
      </div>
    </el-card>

    <el-card shadow="never">
      <template #header>平台运行与变更日志</template>
      <el-table :data="transferLogs" border>
        <el-table-column label="时间" min-width="172">
          <template #default="scope">{{ formatTime(scope.row.created_at) }}</template>
        </el-table-column>
        <el-table-column prop="action" label="动作" min-width="100" />
        <el-table-column prop="mode" label="模式" min-width="100" />
        <el-table-column prop="operator" label="执行人" min-width="120" />
        <el-table-column prop="status" label="状态" min-width="100" />
        <el-table-column prop="message" label="摘要" min-width="220" show-overflow-tooltip />
        <el-table-column prop="detail" label="日志详情" min-width="320" show-overflow-tooltip />
      </el-table>
    </el-card>

    <el-card shadow="never">
      <template #header>操作审计</template>
      <el-table :data="auditLogs" border>
        <el-table-column label="时间" min-width="172">
          <template #default="scope">{{ formatTime(scope.row.created_at) }}</template>
        </el-table-column>
        <el-table-column prop="username" label="用户" min-width="120" />
        <el-table-column prop="method" label="方法" min-width="90" />
        <el-table-column prop="path" label="路径" min-width="240" show-overflow-tooltip />
        <el-table-column prop="operation" label="操作" min-width="180" />
        <el-table-column prop="status" label="状态码" min-width="100" />
        <el-table-column prop="payload" label="请求参数" min-width="320" show-overflow-tooltip />
      </el-table>
    </el-card>
  </div>
</template>

<style scoped>
.logs-page {
  display: grid;
  gap: 16px;
}

.summary-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
}

.summary-label {
  color: #64748b;
}

.summary-value {
  margin-top: 8px;
  font-size: 28px;
  font-weight: 700;
}

.toolbar {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 12px;
}

.tips {
  margin-bottom: 12px;
  color: #475569;
  line-height: 1.8;
}

.service-log-panel {
  display: grid;
  gap: 12px;
}

.service-log-meta {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
  color: #64748b;
}

.service-log-content {
  max-height: 520px;
  overflow: auto;
  margin: 0;
  padding: 16px;
  border-radius: 12px;
  background: #0f172a;
  color: #e2e8f0;
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

@media (max-width: 1200px) {
  .summary-grid {
    grid-template-columns: 1fr;
  }
}
</style>
