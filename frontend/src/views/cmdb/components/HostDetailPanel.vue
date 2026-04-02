<script setup lang="ts">
import type { DependencyItem, Host, HostDetail } from "@/api/cmdb";

defineProps<{
  hosts: Host[];
  selectedHostId: number | null;
  hostDetail: HostDetail | null;
  pageSize: number;
  currentPageHostApps: number;
  currentPageHostOutgoing: number;
  currentPageHostIncoming: number;
  paginatedHostApps: HostDetail["apps"];
  paginatedHostOutgoing: HostDetail["calls_outgoing"];
  paginatedHostIncoming: HostDetail["calls_incoming"];
  formatDependencyEndpoint: (item: DependencyItem, side: "source" | "target") => string;
}>();

const emit = defineEmits<{
  (e: "update:selectedHostId", value: number | null): void;
  (e: "query"): void;
  (e: "update:currentPageHostApps", value: number): void;
  (e: "update:currentPageHostOutgoing", value: number): void;
  (e: "update:currentPageHostIncoming", value: number): void;
}>();
</script>

<template>
  <el-space>
    <el-select
      filterable
      :model-value="selectedHostId"
      placeholder="选择主机"
      style="width: 320px"
      @update:model-value="emit('update:selectedHostId', $event)"
    >
      <el-option v-for="item in hosts" :key="item.ID" :label="item.name" :value="item.ID" />
    </el-select>
    <el-button type="primary" @click="emit('query')">查询</el-button>
  </el-space>

  <el-row class="mt-4" :gutter="16" v-if="hostDetail">
    <el-col :span="8">
      <el-card>
        <b>主机</b>
        <div class="mt-2">{{ hostDetail.host.name }} ({{ hostDetail.host.ip }})</div>
      </el-card>
    </el-col>
    <el-col :span="8">
      <el-card>
        <b>集群</b>
        <div class="mt-2">{{ hostDetail.cluster?.name || "-" }}</div>
      </el-card>
    </el-col>
    <el-col :span="8">
      <el-card>
        <b>域名</b>
        <div class="mt-2">{{ hostDetail.domains.join(", ") || "-" }}</div>
      </el-card>
    </el-col>
    <el-col :span="12" class="mt-4">
      <el-card>
        <b>应用列表</b>
        <div class="mt-2" v-for="app in paginatedHostApps" :key="app.id">
          {{ app.name }} [{{ app.ports.join(",") }}]
        </div>
        <el-pagination
          class="mt-2"
          :current-page="currentPageHostApps"
          :page-size="pageSize"
          :total="hostDetail.apps.length"
          layout="prev, pager, next"
          small
          hide-on-single-page
          @update:current-page="emit('update:currentPageHostApps', $event)"
        />
      </el-card>
    </el-col>
    <el-col :span="12" class="mt-4">
      <el-card>
        <b>调用关系</b>
        <div class="mt-2 text-sm text-gray-500 font-bold">出向：</div>
        <div v-for="(item, idx) in paginatedHostOutgoing" :key="`out-${idx}`">
          {{ formatDependencyEndpoint(item, "source") }} -> {{ formatDependencyEndpoint(item, "target") }} ({{ item.desc || "-" }})
        </div>
        <el-pagination
          class="mt-2"
          :current-page="currentPageHostOutgoing"
          :page-size="pageSize"
          :total="hostDetail.calls_outgoing.length"
          layout="prev, pager, next"
          small
          hide-on-single-page
          @update:current-page="emit('update:currentPageHostOutgoing', $event)"
        />
        <div class="mt-2 text-sm text-gray-500 font-bold">入向：</div>
        <div v-for="(item, idx) in paginatedHostIncoming" :key="`in-${idx}`">
          {{ formatDependencyEndpoint(item, "source") }} -> {{ formatDependencyEndpoint(item, "target") }} ({{ item.desc || "-" }})
        </div>
        <el-pagination
          class="mt-2"
          :current-page="currentPageHostIncoming"
          :page-size="pageSize"
          :total="hostDetail.calls_incoming.length"
          layout="prev, pager, next"
          small
          hide-on-single-page
          @update:current-page="emit('update:currentPageHostIncoming', $event)"
        />
      </el-card>
    </el-col>
  </el-row>
</template>
