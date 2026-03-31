<script setup lang="tsx">
import { ref, onMounted } from "vue";
import { getAuditLogs } from "@/api/system";
import { PureTable } from "@pureadmin/table";
import dayjs from "dayjs";

defineOptions({
  name: "SystemAudit"
});

const loading = ref(true);
const dataList = ref([]);
const columns: TableColumnList = [
  {
    label: "操作描述",
    prop: "operation",
    minWidth: 150
  },
  {
    label: "操作账号",
    prop: "username",
    minWidth: 100
  },
  {
    label: "请求方法",
    prop: "method",
    minWidth: 80,
    cellRenderer: ({ row }) => (
      <el-tag type={row.method === "DELETE" ? "danger" : "info"}>
        {row.method}
      </el-tag>
    )
  },
  {
    label: "请求路径",
    prop: "path",
    minWidth: 200
  },
  {
    label: "操作IP",
    prop: "ip",
    minWidth: 120
  },
  {
    label: "状态码",
    prop: "status",
    cellRenderer: ({ row }) => (
      <el-tag type={row.status >= 400 ? "danger" : "success"}>
        {row.status}
      </el-tag>
    )
  },
  {
    label: "耗时(ms)",
    prop: "duration",
    minWidth: 100
  },
  {
    label: "操作时间",
    prop: "CreatedAt",
    minWidth: 160,
    formatter: ({ CreatedAt }) =>
      dayjs(CreatedAt).format("YYYY-MM-DD HH:mm:ss")
  }
];

async function onSearch() {
  loading.value = true;
  const { code, data } = await getAuditLogs();
  if (code === 0) {
    dataList.value = data.list;
  }
  loading.value = false;
}

onMounted(() => {
  onSearch();
});
</script>

<template>
  <div class="main">
    <div class="bg-bg_color p-4">
      <div class="flex justify-between mb-4">
        <h2 class="text-lg font-bold">全平台操作审计</h2>
        <el-button type="primary" @click="onSearch">刷新日志</el-button>
      </div>
      <pure-table
        border
        adaptive
        align-center
        row-key="id"
        showOverflowTooltip
        :loading="loading"
        :data="dataList"
        :columns="columns"
      />
    </div>
  </div>
</template>

<style scoped>
.main {
  margin: 24px;
}
</style>
