<script setup lang="ts">
import { onMounted, ref } from "vue";
import dayjs from "dayjs";
import { getOpsOverview, type OpsOverviewData } from "@/api/cmdb";
import { message } from "@/utils/message";

defineOptions({
  name: "CMDBOpsOverview"
});

const loading = ref(false);
const overview = ref<OpsOverviewData | null>(null);
const metricMeta: Record<string, { title: string; desc: string }> = {
  clusters: {
    title: "集群登记总数",
    desc: "已录入的集群与环境分组"
  },
  hosts: {
    title: "主机登记总数",
    desc: "纳入 CMDB 管理的服务器节点"
  },
  apps: {
    title: "应用登记总数",
    desc: "挂载在主机上的应用与服务"
  },
  ports: {
    title: "端口登记总数",
    desc: "应用暴露或监听的端口配置"
  },
  domains: {
    title: "域名登记总数",
    desc: "已绑定的访问域名与入口"
  },
  dependencies: {
    title: "依赖链路总数",
    desc: "应用、主机与外部节点之间的调用关系"
  }
};

async function loadOverview() {
  try {
    loading.value = true;
    const resp = await getOpsOverview();
    overview.value = resp.data;
  } catch (error: any) {
    message(error?.message || "加载系统总览失败", { type: "error" });
  } finally {
    loading.value = false;
  }
}

function formatTime(value?: string) {
  if (!value) return "-";
  return dayjs(value).format("YYYY-MM-DD HH:mm:ss");
}

onMounted(loadOverview);
</script>

<template>
  <div class="ops-page" v-loading="loading">
    <div class="ops-grid ops-grid-6" v-if="overview">
      <el-card v-for="(value, key) in overview.counts" :key="key" shadow="hover">
        <div class="metric-label">{{ metricMeta[key]?.title || key }}</div>
        <div class="metric-value">{{ value }}</div>
        <div class="metric-desc">{{ metricMeta[key]?.desc || "-" }}</div>
      </el-card>
    </div>

    <div class="ops-grid ops-grid-2" v-if="overview">
      <el-card shadow="never">
        <template #header>服务健康状态</template>
        <div class="service-list">
          <div v-for="item in overview.services" :key="item.name" class="service-item">
            <div>
              <div class="service-name">{{ item.name }}</div>
              <div class="service-message">{{ item.message }}</div>
            </div>
            <div class="service-meta">
              <el-tag :type="item.status === 'UP' ? 'success' : item.status === 'DEGRADED' ? 'warning' : 'danger'">
                {{ item.status }}
              </el-tag>
              <span>{{ item.latency_ms }} ms</span>
            </div>
          </div>
        </div>
      </el-card>

      <el-card shadow="never">
        <template #header>资源概览</template>
        <div class="resource-box">
          <div class="resource-item">
            <span>总 CPU</span>
            <strong>{{ overview.resource_totals.total_cpu }}</strong>
          </div>
          <div class="resource-item">
            <span>总内存</span>
            <strong>{{ overview.resource_totals.total_memory }}</strong>
          </div>
          <div class="resource-item">
            <span>在线主机</span>
            <strong>{{ overview.resource_totals.online_hosts }}</strong>
          </div>
          <div class="resource-item">
            <span>离线主机</span>
            <strong>{{ overview.resource_totals.offline_hosts }}</strong>
          </div>
        </div>
      </el-card>
    </div>

    <div class="ops-grid ops-grid-2" v-if="overview">
      <el-card shadow="never">
        <template #header>应用类型分布</template>
        <div class="dist-list">
          <div v-for="item in overview.app_types" :key="item.name" class="dist-row">
            <span>{{ item.name }}</span>
            <el-tag>{{ item.value }}</el-tag>
          </div>
        </div>
      </el-card>

      <el-card shadow="never">
        <template #header>部署方式分布</template>
        <div class="dist-list">
          <div v-for="item in overview.deploy_types" :key="item.name" class="dist-row">
            <span>{{ item.name }}</span>
            <el-tag type="info">{{ item.value }}</el-tag>
          </div>
        </div>
      </el-card>
    </div>

    <div class="ops-grid ops-grid-2" v-if="overview">
      <el-card shadow="never">
        <template #header>最近变更应用</template>
        <el-table :data="overview.latest_apps" border>
          <el-table-column prop="name" label="应用" min-width="160" />
          <el-table-column prop="host_name" label="主机" min-width="140" />
          <el-table-column prop="type" label="类型" min-width="120" />
          <el-table-column prop="deploy_type" label="部署方式" min-width="120" />
          <el-table-column label="更新时间" min-width="180">
            <template #default="scope">{{ formatTime(scope.row.updated_at) }}</template>
          </el-table-column>
        </el-table>
      </el-card>

      <el-card shadow="never">
        <template #header>最近变更域名 / 依赖</template>
        <div class="stack-list">
          <div class="stack-section">
            <div class="stack-title">域名</div>
            <div v-for="item in overview.latest_domains" :key="`domain-${item.id}`" class="stack-item">
              <div>{{ item.domain }}</div>
              <div class="stack-meta">{{ item.app_name || item.host_name || "-" }} · {{ formatTime(item.updated_at) }}</div>
            </div>
          </div>
          <div class="stack-section">
            <div class="stack-title">依赖</div>
            <div v-for="item in overview.latest_dependencies" :key="`dep-${item.id}`" class="stack-item">
              <div>{{ item.source }} -> {{ item.target }}</div>
              <div class="stack-meta">{{ item.desc || "-" }} · {{ formatTime(item.updated_at) }}</div>
            </div>
          </div>
        </div>
      </el-card>
    </div>
  </div>
</template>

<style scoped>
.ops-page {
  display: grid;
  gap: 16px;
}

.ops-grid {
  display: grid;
  gap: 16px;
}

.ops-grid-6 {
  grid-template-columns: repeat(6, minmax(0, 1fr));
}

.ops-grid-2 {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.metric-label {
  color: #64748b;
}

.metric-value {
  margin-top: 8px;
  font-size: 28px;
  font-weight: 700;
  color: #0f172a;
}

.metric-desc {
  margin-top: 8px;
  color: #94a3b8;
  font-size: 12px;
  line-height: 1.6;
}

.service-list,
.dist-list,
.stack-list {
  display: grid;
  gap: 12px;
}

.service-item,
.dist-row,
.resource-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.service-name,
.stack-title {
  font-weight: 600;
  color: #0f172a;
}

.service-message,
.stack-meta {
  margin-top: 4px;
  color: #64748b;
  font-size: 12px;
}

.service-meta {
  display: flex;
  align-items: center;
  gap: 10px;
  color: #475569;
}

.resource-box {
  display: grid;
  gap: 14px;
}

.stack-item {
  padding: 10px 12px;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
}

@media (max-width: 1200px) {
  .ops-grid-6,
  .ops-grid-2 {
    grid-template-columns: 1fr;
  }
}
</style>
