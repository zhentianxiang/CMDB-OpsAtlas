<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import dayjs from "dayjs";
import {
  executeSqlConsole,
  querySqlConsole,
  listTransferRecords,
  type SqlConsoleResult,
  type TransferRecordItem
} from "@/api/cmdb";
import { hasPerms } from "@/utils/auth";
import { message } from "@/utils/message";

defineOptions({
  name: "OpsSqlConsole"
});

const mode = ref<"query" | "execute">("query");
const loading = ref(false);
const sql = ref("SELECT id, name, created_at FROM clusters ORDER BY id DESC LIMIT 20");
const result = ref<SqlConsoleResult | null>(null);
const history = ref<TransferRecordItem[]>([]);
const confirmVisible = ref(false);
const confirmKeyword = ref("");

const canExecuteWrite = computed(() => hasPerms("ops:sql:execute"));
const canConfirmExecute = computed(() => confirmKeyword.value.trim().toUpperCase() === "EXECUTE");

async function loadHistory() {
  try {
    const resp = await listTransferRecords("sql");
    history.value = resp.data || [];
  } catch (error: any) {
    message(error?.message || "加载 SQL 历史失败", { type: "error" });
  }
}

async function runSql() {
  if (!sql.value.trim()) {
    message("请输入 SQL 语句", { type: "warning" });
    return;
  }

  if (mode.value === "execute" && !canExecuteWrite.value) {
    message("当前账号没有 SQL 执行权限", { type: "error" });
    return;
  }

  if (mode.value === "execute") {
    confirmKeyword.value = "";
    confirmVisible.value = true;
    return;
  }

  await submitSql();
}

async function submitSql() {
  try {
    loading.value = true;
    const resp =
      mode.value === "query"
        ? await querySqlConsole({ sql: sql.value })
        : await executeSqlConsole({ sql: sql.value });
    result.value = resp.data;
    message(mode.value === "query" ? "SQL 查询完成" : "SQL 执行完成", {
      type: "success"
    });
    await loadHistory();
  } catch (error: any) {
    message(error?.message || "SQL 执行失败", { type: "error" });
  } finally {
    loading.value = false;
  }
}

async function confirmExecute() {
  if (!canConfirmExecute.value) return;
  confirmVisible.value = false;
  await submitSql();
}

function applyExample(value: string, nextMode: "query" | "execute") {
  mode.value = nextMode;
  sql.value = value;
}

function replayHistory(item: TransferRecordItem) {
  sql.value = item.detail || "";
  mode.value = item.mode === "execute" ? "execute" : "query";
}

function formatTime(value?: string) {
  return value ? dayjs(value).format("YYYY-MM-DD HH:mm:ss") : "-";
}

onMounted(loadHistory);
</script>

<template>
  <div class="sql-page" v-loading="loading">
    <el-card shadow="never">
      <template #header>SQL 控制台</template>
      <div class="tips">
        <p>当前控制台连接的是 `cmdb_resource` 库，适合做快速排查和受控变更。</p>
        <p>只读角色仅可执行查询模式；执行模式要求 `ops:sql:execute` 权限，并且需要二次确认。</p>
      </div>

      <el-space wrap class="mb-4">
        <el-radio-group v-model="mode">
          <el-radio-button label="query">查询模式</el-radio-button>
          <el-radio-button label="execute" :disabled="!canExecuteWrite">执行模式</el-radio-button>
        </el-radio-group>
        <el-button @click="applyExample('SHOW TABLES', 'query')">示例: SHOW TABLES</el-button>
        <el-button @click="applyExample('SELECT id, name, status FROM hosts ORDER BY id DESC LIMIT 20', 'query')">
          示例: 查询主机
        </el-button>
        <el-button
          v-if="canExecuteWrite"
          @click="applyExample(`UPDATE hosts SET status = 'offline' WHERE id = 1`, 'execute')"
        >
          示例: 更新状态
        </el-button>
      </el-space>

      <el-input
        v-model="sql"
        type="textarea"
        :rows="10"
        placeholder="请输入单条 SQL 语句"
      />

      <div class="mt-4 action-row">
        <el-button type="primary" @click="runSql">
          {{ mode === "query" ? "执行查询" : "执行变更" }}
        </el-button>
      </div>
    </el-card>

    <el-card v-if="result" shadow="never">
      <template #header>执行结果</template>
      <div class="result-summary">
        <span>语句类型：{{ result.statement_type }}</span>
        <span>耗时：{{ result.elapsed_ms }} ms</span>
        <span>影响行数：{{ result.affected_rows }}</span>
        <span>结果行数：{{ result.row_count }}</span>
        <span v-if="result.truncated">结果过长，已截断为前 200 行</span>
      </div>

      <el-table v-if="result.columns?.length" :data="result.rows" border>
        <el-table-column
          v-for="column in result.columns"
          :key="column"
          :prop="column"
          :label="column"
          min-width="160"
          show-overflow-tooltip
        />
      </el-table>
    </el-card>

    <el-card shadow="never">
      <template #header>执行历史回放</template>
      <el-table :data="history" border>
        <el-table-column label="时间" min-width="172">
          <template #default="scope">{{ formatTime(scope.row.created_at) }}</template>
        </el-table-column>
        <el-table-column prop="mode" label="模式" min-width="100" />
        <el-table-column prop="operator" label="执行人" min-width="120" />
        <el-table-column prop="status" label="状态" min-width="100" />
        <el-table-column prop="message" label="结果摘要" min-width="220" show-overflow-tooltip />
        <el-table-column prop="detail" label="SQL 内容" min-width="360" show-overflow-tooltip />
        <el-table-column label="操作" width="120">
          <template #default="scope">
            <el-button link type="primary" @click="replayHistory(scope.row)">回放</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="confirmVisible" title="确认执行 SQL" width="620px">
      <div class="confirm-box">
        <p>执行模式会直接修改数据库。</p>
        <p>请确认 SQL 正确，并输入 `EXECUTE` 继续。</p>
        <pre class="confirm-sql">{{ sql }}</pre>
        <el-input v-model="confirmKeyword" placeholder="请输入 EXECUTE" />
      </div>
      <template #footer>
        <el-button @click="confirmVisible = false">取消</el-button>
        <el-button type="danger" :disabled="!canConfirmExecute" @click="confirmExecute">
          确认执行
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.sql-page {
  display: grid;
  gap: 16px;
}

.tips {
  margin-bottom: 12px;
  color: #475569;
  line-height: 1.8;
}

.action-row {
  display: flex;
  gap: 12px;
}

.result-summary {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
  margin-bottom: 12px;
  color: #475569;
}

.confirm-box {
  display: grid;
  gap: 12px;
  color: #475569;
}

.confirm-sql {
  max-height: 240px;
  overflow: auto;
  margin: 0;
  padding: 16px;
  border-radius: 12px;
  background: #0f172a;
  color: #e2e8f0;
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
