<script setup lang="ts">
import type { AppItem, Cluster, DomainItem, Host } from "@/api/cmdb";

defineProps<{
  clusters: Cluster[];
  topologyHostOptions: Host[];
  topologyAppOptions: AppItem[];
  topologyDomainOptions: DomainItem[];
  selectedClusterId: number | null;
  selectedTopologyHostId: number | null;
  selectedTopologyAppId: number | null;
  selectedTopologyDomainId: number | null;
}>();

const emit = defineEmits<{
  (e: "update:selectedClusterId", value: number | null): void;
  (e: "update:selectedTopologyHostId", value: number | null): void;
  (e: "update:selectedTopologyAppId", value: number | null): void;
  (e: "update:selectedTopologyDomainId", value: number | null): void;
  (e: "query"): void;
  (e: "reset"): void;
}>();
</script>

<template>
  <div class="mb-3 flex flex-wrap items-center gap-3">
    <el-select
      filterable
      :model-value="selectedClusterId"
      clearable
      placeholder="按集群筛选"
      style="width: 320px"
      @update:model-value="emit('update:selectedClusterId', $event)"
    >
      <el-option v-for="item in clusters" :key="item.ID" :label="item.name" :value="item.ID" />
    </el-select>
    <el-select
      filterable
      :model-value="selectedTopologyHostId"
      clearable
      placeholder="按主机筛选链路"
      style="width: 320px"
      @update:model-value="emit('update:selectedTopologyHostId', $event)"
    >
      <el-option
        v-for="item in topologyHostOptions"
        :key="item.ID"
        :label="`${item.name} · ${item.private_ip || item.public_ip || item.ip || '-'}`"
        :value="item.ID"
      />
    </el-select>
    <el-select
      filterable
      :model-value="selectedTopologyAppId"
      clearable
      placeholder="按应用筛选链路"
      style="width: 320px"
      @update:model-value="emit('update:selectedTopologyAppId', $event)"
    >
      <el-option
        v-for="item in topologyAppOptions"
        :key="item.ID"
        :label="`${item.name} · ${item.type || '未分类'}`"
        :value="item.ID"
      />
    </el-select>
    <el-select
      filterable
      :model-value="selectedTopologyDomainId"
      clearable
      placeholder="按域名筛选链路"
      style="width: 360px"
      @update:model-value="emit('update:selectedTopologyDomainId', $event)"
    >
      <el-option v-for="item in topologyDomainOptions" :key="item.ID" :label="item.domain" :value="item.ID" />
    </el-select>
    <el-button type="primary" @click="emit('query')">加载拓扑</el-button>
    <el-button @click="emit('reset')">清空筛选</el-button>
    <el-tag type="info">支持拖拽节点、滚轮缩放、点击节点查看详情</el-tag>
  </div>
</template>
