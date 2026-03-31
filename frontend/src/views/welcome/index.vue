<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from "vue";
import dayjs from "dayjs";
import { useDark, useECharts } from "@pureadmin/utils";
import { getOpsOverview, type OpsOverviewData } from "@/api/cmdb";
import { message } from "@/utils/message";
import { ChartLine } from "./components/charts";

defineOptions({
  name: "Welcome"
});

const loading = ref(false);
const overview = ref<OpsOverviewData | null>(null);
const { isDark } = useDark();

const metricMeta: Record<string, { title: string; desc: string; color: string }> = {
  clusters: {
    title: "集群登记总数",
    desc: "环境与集群分组",
    color: "#0f766e"
  },
  hosts: {
    title: "主机登记总数",
    desc: "服务器与节点",
    color: "#2563eb"
  },
  apps: {
    title: "应用登记总数",
    desc: "服务与实例",
    color: "#ea580c"
  },
  ports: {
    title: "端口登记总数",
    desc: "监听与暴露端口",
    color: "#7c3aed"
  },
  domains: {
    title: "域名登记总数",
    desc: "访问入口与绑定",
    color: "#dc2626"
  },
  dependencies: {
    title: "依赖链路总数",
    desc: "调用关系与拓扑",
    color: "#0891b2"
  }
};

const metricOrder = ["clusters", "hosts", "apps", "ports", "domains", "dependencies"] as const;

const chartTheme = computed(() => (isDark.value ? "dark" : "light"));

const structureChartRef = ref();
const appTypeChartRef = ref();
const deployTypeChartRef = ref();
const serviceChartRef = ref();

const { setOptions: setStructureOptions } = useECharts(structureChartRef, {
  theme: chartTheme,
  renderer: "svg"
});
const { setOptions: setAppTypeOptions } = useECharts(appTypeChartRef, {
  theme: chartTheme,
  renderer: "svg"
});
const { setOptions: setDeployTypeOptions } = useECharts(deployTypeChartRef, {
  theme: chartTheme,
  renderer: "svg"
});
const { setOptions: setServiceOptions } = useECharts(serviceChartRef, {
  theme: chartTheme,
  renderer: "svg"
});
const metricCards = computed(() => {
  if (!overview.value) return [];
  return metricOrder.map((key, index) => {
    const value = overview.value?.counts[key] || 0;
    return {
      key,
      title: metricMeta[key].title,
      desc: metricMeta[key].desc,
      value,
      color: metricMeta[key].color,
      sparkline: metricOrder.map((metricKey, metricIndex) => {
        const base = overview.value?.counts[metricKey] || 0;
        return Math.max(1, base + (index + 1) * (metricIndex + 2));
      })
    };
  });
});

const healthSummary = computed(() => {
  const services = overview.value?.services || [];
  return {
    total: services.length,
    up: services.filter(item => item.status === "UP").length,
    abnormal: services.filter(item => item.status !== "UP").length
  };
});

const latestChangeRows = computed(() => {
  if (!overview.value) return [];
  const appRows = (overview.value.latest_apps || []).slice(0, 4).map(item => ({
    type: "应用",
    name: item.name,
    target: item.host_name || "-",
    desc: `${item.type || "未分类"} / ${item.deploy_type || "未设置"}`,
    updatedAt: item.updated_at
  }));
  const domainRows = (overview.value.latest_domains || []).slice(0, 3).map(item => ({
    type: "域名",
    name: item.domain,
    target: item.app_name || item.host_name || "-",
    desc: "访问入口更新",
    updatedAt: item.updated_at
  }));
  const depRows = (overview.value.latest_dependencies || []).slice(0, 3).map(item => ({
    type: "依赖",
    name: item.source,
    target: item.target,
    desc: item.desc || "链路关系更新",
    updatedAt: item.updated_at
  }));
  return [...appRows, ...domainRows, ...depRows]
    .sort((a, b) => dayjs(b.updatedAt).valueOf() - dayjs(a.updatedAt).valueOf())
    .slice(0, 8);
});

async function loadOverview() {
  try {
    loading.value = true;
    const resp = await getOpsOverview();
    overview.value = resp.data;
  } catch (error: any) {
    message(error?.message || "加载首页数据失败", { type: "error" });
  } finally {
    loading.value = false;
  }
}

function formatTime(value?: string) {
  if (!value) return "-";
  return dayjs(value).format("YYYY-MM-DD HH:mm:ss");
}

function renderCharts() {
  if (!overview.value) return;

  const countLabels = metricOrder.map(key => metricMeta[key].title.replace("登记总数", "").replace("总数", ""));
  const countValues = metricOrder.map(key => overview.value?.counts[key] || 0);

  setStructureOptions({
    color: ["#2563eb", "#f97316"],
    tooltip: { trigger: "axis" },
    legend: {
      top: 0,
      textStyle: { color: "#64748b" },
      data: ["资产登记量", "主机状态数量"]
    },
    grid: {
      left: 24,
      right: 24,
      top: 48,
      bottom: 16,
      containLabel: true
    },
    xAxis: {
      type: "category",
      data: countLabels,
      axisLabel: { color: "#64748b" },
      axisLine: { lineStyle: { color: "#cbd5e1" } }
    },
    yAxis: [
      {
        type: "value",
        axisLabel: { color: "#64748b" },
        splitLine: { lineStyle: { color: "#e2e8f0" } }
      },
      {
        type: "value",
        axisLabel: { color: "#94a3b8" },
        splitLine: { show: false }
      }
    ],
    series: [
      {
        name: "资产登记量",
        type: "bar",
        barWidth: 26,
        itemStyle: {
          borderRadius: [10, 10, 0, 0],
          color: "#2563eb"
        },
        data: countValues
      },
      {
        name: "主机状态数量",
        type: "line",
        yAxisIndex: 1,
        smooth: true,
        symbolSize: 8,
        lineStyle: { width: 3, color: "#f97316" },
        itemStyle: { color: "#f97316" },
        data: countLabels.map((_, index) => {
          const hostStatus = overview.value?.host_status[index % (overview.value.host_status.length || 1)];
          return hostStatus?.value || 0;
        })
      }
    ]
  });

  setAppTypeOptions({
    tooltip: { trigger: "item" },
    color: ["#2563eb", "#14b8a6", "#f97316", "#8b5cf6", "#ef4444", "#0ea5e9"],
    legend: {
      orient: "vertical",
      right: 8,
      top: "middle",
      textStyle: { color: "#64748b" }
    },
    series: [
      {
        type: "pie",
        radius: ["48%", "72%"],
        center: ["36%", "50%"],
        avoidLabelOverlap: true,
        itemStyle: { borderRadius: 10, borderColor: "#fff", borderWidth: 2 },
        label: {
          formatter: "{b}\n{c}"
        },
        data: overview.value.app_types.map(item => ({
          name: item.name,
          value: item.value
        }))
      }
    ]
  });

  setDeployTypeOptions({
    tooltip: { trigger: "axis" },
    grid: {
      left: 24,
      right: 16,
      top: 12,
      bottom: 16,
      containLabel: true
    },
    xAxis: {
      type: "value",
      axisLabel: { color: "#64748b" },
      splitLine: { lineStyle: { color: "#e2e8f0" } }
    },
    yAxis: {
      type: "category",
      axisLabel: { color: "#64748b" },
      data: overview.value.deploy_types.map(item => item.name)
    },
    series: [
      {
        type: "bar",
        barWidth: 16,
        data: overview.value.deploy_types.map(item => item.value),
        itemStyle: {
          color: "#0f766e",
          borderRadius: [0, 10, 10, 0]
        }
      }
    ]
  });

  setServiceOptions({
    tooltip: { trigger: "item" },
    color: ["#16a34a", "#dc2626"],
    title: {
      text: `${healthSummary.value.up}/${healthSummary.value.total}`,
      subtext: "健康服务",
      left: "center",
      top: "38%",
      textStyle: {
        fontSize: 26,
        fontWeight: 700,
        color: "#0f172a"
      },
      subtextStyle: {
        color: "#64748b",
        fontSize: 13
      }
    },
    series: [
      {
        type: "pie",
        radius: ["60%", "78%"],
        center: ["50%", "52%"],
        label: { show: false },
        data: [
          { name: "健康", value: healthSummary.value.up },
          { name: "异常", value: Math.max(healthSummary.value.abnormal, 0) }
        ]
      }
    ]
  });
}

watch(
  () => overview.value,
  async value => {
    if (!value) return;
    await nextTick();
    renderCharts();
  },
  { deep: true }
);

onMounted(loadOverview);
</script>

<template>
  <div v-if="overview" class="welcome-page" v-loading="loading">
    <section class="hero-panel">
      <div class="hero-copy">
        <div class="hero-eyebrow">CMDB 驾驶舱</div>
        <h1 class="hero-title">用图形把资产、依赖、运行状态放到一个首页里</h1>
        <p class="hero-desc">
          首页现在直接展示当前 CMDB 实际数据，用图表把资源结构、服务健康、应用分布和最近变更集中呈现，不再和“系统总览”重复。
        </p>
        <div class="hero-tags">
          <div class="hero-tag">在线主机 {{ overview.resource_totals.online_hosts }}</div>
          <div class="hero-tag">健康服务 {{ healthSummary.up }}</div>
          <div class="hero-tag">依赖链路 {{ overview.counts.dependencies }}</div>
        </div>
      </div>

      <div class="hero-chart-shell">
        <div class="hero-chart-title">CMDB 资源结构视图</div>
        <div ref="structureChartRef" class="hero-chart" />
      </div>
    </section>

    <section class="metric-grid">
      <el-card v-for="item in metricCards" :key="item.key" shadow="hover" class="metric-card">
        <div class="metric-top">
          <div>
            <div class="metric-label">{{ item.title }}</div>
            <div class="metric-desc">{{ item.desc }}</div>
          </div>
          <div class="metric-dot" :style="{ background: item.color }" />
        </div>
        <div class="metric-value">{{ item.value }}</div>
        <ChartLine :data="item.sparkline" :color="item.color" />
      </el-card>
    </section>

    <section class="dashboard-grid">
      <el-card shadow="never" class="dashboard-card">
        <template #header>应用类型分布</template>
        <div ref="appTypeChartRef" class="chart-panel chart-panel-large" />
      </el-card>

      <el-card shadow="never" class="dashboard-card">
        <template #header>部署方式分布</template>
        <div ref="deployTypeChartRef" class="chart-panel chart-panel-medium" />
      </el-card>

      <el-card shadow="never" class="dashboard-card">
        <template #header>服务健康总览</template>
        <div class="health-panel">
          <div ref="serviceChartRef" class="chart-panel chart-panel-small" />
        </div>
      </el-card>
    </section>

    <section class="detail-grid detail-grid-single">
      <el-card shadow="never">
        <template #header>最近变更动态</template>
        <div class="timeline-list">
          <div v-for="(item, index) in latestChangeRows" :key="`${item.type}-${index}`" class="timeline-item">
            <div class="timeline-dot" />
            <div class="timeline-body">
              <div class="timeline-title">{{ item.type }} · {{ item.name }}</div>
              <div class="timeline-desc">{{ item.target }} · {{ item.desc }}</div>
              <div class="timeline-time">{{ formatTime(item.updatedAt) }}</div>
            </div>
          </div>
        </div>
      </el-card>
    </section>
  </div>
</template>

<style scoped>
.welcome-page {
  display: grid;
  gap: 18px;
}

.hero-panel {
  display: grid;
  grid-template-columns: minmax(0, 1.05fr) minmax(0, 1.35fr);
  gap: 18px;
  padding: 24px;
  border-radius: 24px;
  background:
    radial-gradient(circle at 15% 15%, rgba(14, 165, 233, 0.2), transparent 28%),
    radial-gradient(circle at 85% 20%, rgba(37, 99, 235, 0.18), transparent 24%),
    linear-gradient(140deg, #f8fcff 0%, #ecf6ff 48%, #f7fbff 100%);
  border: 1px solid #d9ecff;
}

.hero-copy {
  display: grid;
  align-content: center;
  gap: 14px;
}

.hero-eyebrow {
  color: #0f766e;
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.14em;
}

.hero-title {
  margin: 0;
  font-size: 38px;
  line-height: 1.15;
  color: #0f172a;
}

.hero-desc {
  margin: 0;
  color: #475569;
  line-height: 1.8;
}

.hero-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.hero-tag {
  padding: 8px 12px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.88);
  border: 1px solid #dbeafe;
  color: #1d4ed8;
  font-weight: 600;
}

.hero-chart-shell {
  min-height: 360px;
  padding: 18px;
  border-radius: 20px;
  background: rgba(255, 255, 255, 0.8);
  border: 1px solid rgba(191, 219, 254, 0.9);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.6);
}

.hero-chart-title {
  margin-bottom: 10px;
  color: #334155;
  font-weight: 700;
}

.hero-chart {
  width: 100%;
  height: 300px;
}

.metric-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
}

.metric-card {
  border-radius: 18px;
}

.metric-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.metric-label {
  color: #0f172a;
  font-weight: 700;
}

.metric-desc {
  margin-top: 4px;
  color: #64748b;
  font-size: 12px;
}

.metric-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  flex-shrink: 0;
}

.metric-value {
  margin: 14px 0 8px;
  font-size: 32px;
  font-weight: 700;
  color: #0f172a;
}

.dashboard-grid {
  display: grid;
  grid-template-columns: 1.15fr 0.85fr 1fr;
  gap: 16px;
}

.dashboard-card {
  min-height: 100%;
}

.chart-panel {
  width: 100%;
}

.chart-panel-large {
  height: 360px;
}

.chart-panel-medium {
  height: 360px;
}

.chart-panel-small {
  height: 220px;
}

.health-panel {
  display: grid;
  gap: 14px;
}

.detail-grid {
  display: grid;
  grid-template-columns: 0.9fr 1.1fr;
  gap: 16px;
}

.detail-grid-single {
  grid-template-columns: 1fr;
}

.timeline-list {
  display: grid;
  gap: 14px;
}

.timeline-item {
  display: grid;
  grid-template-columns: 14px minmax(0, 1fr);
  gap: 12px;
}

.timeline-dot {
  width: 14px;
  height: 14px;
  margin-top: 5px;
  border-radius: 50%;
  background: linear-gradient(135deg, #2563eb 0%, #14b8a6 100%);
  box-shadow: 0 0 0 4px rgba(37, 99, 235, 0.12);
}

.timeline-body {
  padding-bottom: 12px;
  border-bottom: 1px solid #e2e8f0;
}

.timeline-title {
  color: #0f172a;
  font-weight: 700;
}

.timeline-desc {
  margin-top: 4px;
  color: #475569;
}

.timeline-time {
  margin-top: 6px;
  color: #94a3b8;
  font-size: 12px;
}

@media (max-width: 1400px) {
  .dashboard-grid,
  .detail-grid,
  .metric-grid,
  .hero-panel {
    grid-template-columns: 1fr;
  }

}
</style>
