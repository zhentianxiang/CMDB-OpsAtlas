<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import dayjs from "dayjs";
import { listBackupFiles, listTransferRecords, type BackupFileItem, type TransferRecordItem } from "@/api/cmdb";
import { hasPerms } from "@/utils/auth";
import { message } from "@/utils/message";

defineOptions({
  name: "OpsTasks"
});

const loading = ref(false);
const records = ref<TransferRecordItem[]>([]);
const backupFiles = ref<BackupFileItem[]>([]);

const taskCards = computed(() => [
  { label: "最近任务", value: records.value.length },
  { label: "备份文件", value: backupFiles.value.length },
  {
    label: "失败任务",
    value: records.value.filter(item => item.status !== "success").length
  }
]);

async function loadData() {
  try {
    loading.value = true;
    const [recordsResp, filesResp] = await Promise.all([
      listTransferRecords(),
      hasPerms("ops:backup:view") ? listBackupFiles() : Promise.resolve({ data: [] })
    ]);
    records.value = recordsResp.data || [];
    backupFiles.value = filesResp.data || [];
  } catch (error: any) {
    message(error?.message || "加载任务中心失败", { type: "error" });
  } finally {
    loading.value = false;
  }
}

function formatTime(value?: string) {
  return value ? dayjs(value).format("YYYY-MM-DD HH:mm:ss") : "-";
}

onMounted(loadData);
</script>

<template>
  <div class="ops-page" v-loading="loading">
    <div class="ops-cards">
      <el-card v-for="item in taskCards" :key="item.label" shadow="hover">
        <div class="card-label">{{ item.label }}</div>
        <div class="card-value">{{ item.value }}</div>
      </el-card>
    </div>

    <el-card shadow="never">
      <template #header>最近运维任务</template>
      <el-table :data="records" border>
        <el-table-column label="时间" min-width="172">
          <template #default="scope">{{ formatTime(scope.row.created_at) }}</template>
        </el-table-column>
        <el-table-column prop="action" label="动作" min-width="100" />
        <el-table-column prop="mode" label="模式" min-width="100" />
        <el-table-column prop="operator" label="执行人" min-width="120" />
        <el-table-column label="状态" width="100">
          <template #default="scope">
            <el-tag :type="scope.row.status === 'success' ? 'success' : scope.row.status === 'running' ? 'warning' : 'danger'">
              {{ scope.row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="message" label="结果摘要" min-width="220" show-overflow-tooltip />
        <el-table-column prop="detail" label="详情" min-width="280" show-overflow-tooltip />
      </el-table>
    </el-card>

    <el-card shadow="never">
      <template #header>备份产物</template>
      <el-table :data="backupFiles" border>
        <el-table-column prop="filename" label="文件名" min-width="240" show-overflow-tooltip />
        <el-table-column prop="backup_type" label="备份类型" min-width="120" />
        <el-table-column prop="trigger_source" label="触发方式" min-width="120" />
        <el-table-column prop="operator" label="执行人" min-width="120" />
        <el-table-column label="开始时间" min-width="172">
          <template #default="scope">{{ formatTime(scope.row.started_at) }}</template>
        </el-table-column>
        <el-table-column label="完成时间" min-width="172">
          <template #default="scope">{{ formatTime(scope.row.completed_at) }}</template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<style scoped>
.ops-page {
  display: grid;
  gap: 16px;
}

.ops-cards {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
}

.card-label {
  color: #64748b;
}

.card-value {
  margin-top: 8px;
  font-size: 28px;
  font-weight: 700;
}

@media (max-width: 1200px) {
  .ops-cards {
    grid-template-columns: 1fr;
  }
}
</style>
