<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { getOpsOverview, type OpsOverviewData } from "@/api/cmdb";
import { message } from "@/utils/message";

defineOptions({
  name: "OpsRuntime"
});

const loading = ref(false);
const overview = ref<OpsOverviewData | null>(null);

const unhealthyCount = computed(
  () => overview.value?.services.filter(item => item.status !== "UP").length || 0
);

async function loadOverview() {
  try {
    loading.value = true;
    const overviewResp = await getOpsOverview();
    overview.value = overviewResp.data;
  } catch (error: any) {
    message(error?.message || "加载运行状态失败", { type: "error" });
  } finally {
    loading.value = false;
  }
}

onMounted(loadOverview);
</script>

<template>
  <div class="runtime-page" v-loading="loading" v-if="overview">
    <div class="runtime-top">
      <el-card shadow="hover">
        <div class="title">在线服务</div>
        <div class="value">{{ overview.services.filter(item => item.status === "UP").length }}</div>
      </el-card>
      <el-card shadow="hover">
        <div class="title">异常服务</div>
        <div class="value warn">{{ unhealthyCount }}</div>
      </el-card>
      <el-card shadow="hover">
        <div class="title">在线主机</div>
        <div class="value">{{ overview.resource_totals.online_hosts }}</div>
      </el-card>
      <el-card shadow="hover">
        <div class="title">登记 CPU</div>
        <div class="value">{{ overview.resource_totals.total_cpu }}</div>
      </el-card>
    </div>

    <el-card shadow="never">
      <template #header>微服务运行状态</template>
      <el-table :data="overview.services" border>
        <el-table-column prop="name" label="服务" min-width="180" />
        <el-table-column label="状态" width="120">
          <template #default="scope">
            <el-tag :type="scope.row.status === 'UP' ? 'success' : scope.row.status === 'DEGRADED' ? 'warning' : 'danger'">
              {{ scope.row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="latency_ms" label="耗时(ms)" width="120" />
        <el-table-column prop="message" label="信息" min-width="220" />
        <el-table-column prop="url" label="健康检查地址" min-width="280" />
      </el-table>
    </el-card>

    <div class="runtime-grid">
      <el-card shadow="never">
        <template #header>主机状态分布</template>
        <div class="list">
          <div v-for="item in overview.host_status" :key="item.name" class="row">
            <span>{{ item.name }}</span>
            <strong>{{ item.value }}</strong>
          </div>
        </div>
      </el-card>

      <el-card shadow="never">
        <template #header>说明</template>
        <div class="note">
          <p>当前页面的“运行状态”来自微服务健康检查和 CMDB 已登记资源统计。</p>
          <p>更详细的运维日志请统一在日志中心查看；若要接入容器 stdout/stderr、文件日志或主机系统日志，可以继续对接 Loki、ELK 或日志 Agent。</p>
        </div>
      </el-card>
    </div>
  </div>
</template>

<style scoped>
.runtime-page,
.runtime-grid,
.list {
  display: grid;
  gap: 16px;
}

.runtime-top {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
}

.title {
  color: #64748b;
}

.value {
  margin-top: 8px;
  font-size: 28px;
  font-weight: 700;
}

.value.warn {
  color: #dc2626;
}

.runtime-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.row {
  display: flex;
  justify-content: space-between;
  padding: 10px 0;
  border-bottom: 1px solid #e2e8f0;
}

.note {
  color: #475569;
  line-height: 1.8;
}

@media (max-width: 1200px) {
  .runtime-top,
  .runtime-grid {
    grid-template-columns: 1fr;
  }
}
</style>
