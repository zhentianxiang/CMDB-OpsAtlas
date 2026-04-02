<script setup lang="ts">
import { ref } from "vue";
import type { TableInstance } from "element-plus";
import type { CmdbListState, ResourceConfig } from "../schema";
import { formatValue, getColumnLabel } from "../display";

defineProps<{
  activeConfig: ResourceConfig;
  currentSearchKeyword: string;
  allActiveRows: any[];
  activeRows: any[];
  listState: CmdbListState;
  canUpdateCurrent: boolean;
  canDeleteCurrent: boolean;
  currentPage: number;
  pageSize: number;
}>();

const emit = defineEmits<{
  (e: "selection-change", rows: any[]): void;
  (e: "edit", row: any): void;
  (e: "delete", row: any): void;
  (e: "update:currentPage", value: number): void;
  (e: "update:pageSize", value: number): void;
}>();

const tableRef = ref<TableInstance>();

defineExpose({
  clearSelection() {
    tableRef.value?.clearSelection();
  }
});
</script>

<template>
  <div v-if="currentSearchKeyword.trim()" class="table-hint">
    搜索“{{ currentSearchKeyword.trim() }}”命中 {{ allActiveRows.length }} 条
  </div>
  <el-table ref="tableRef" :data="activeRows" border row-key="ID" @selection-change="emit('selection-change', $event)">
    <el-table-column type="selection" width="48" align="center" />
    <el-table-column
      v-for="column in activeConfig.columns"
      :key="column"
      :prop="column"
      :label="getColumnLabel(column)"
      min-width="120"
    >
      <template #default="scope">
        {{ formatValue(activeConfig.key, column, column === "address" ? scope.row : scope.row[column], listState) }}
      </template>
    </el-table-column>
    <el-table-column label="操作" fixed="right" width="160">
      <template #default="scope">
        <el-button v-if="canUpdateCurrent" link type="primary" @click="emit('edit', scope.row)">编辑</el-button>
        <el-popconfirm v-if="canDeleteCurrent" title="确认删除?" @confirm="emit('delete', scope.row)">
          <template #reference>
            <el-button link type="danger">删除</el-button>
          </template>
        </el-popconfirm>
      </template>
    </el-table-column>
  </el-table>
  <div class="mt-4 flex justify-end">
    <el-pagination
      :current-page="currentPage"
      :page-size="pageSize"
      :total="allActiveRows.length"
      :page-sizes="[10, 20, 50, 100]"
      layout="total, sizes, prev, pager, next, jumper"
      @update:current-page="emit('update:currentPage', $event)"
      @update:page-size="emit('update:pageSize', $event)"
    />
  </div>
</template>
