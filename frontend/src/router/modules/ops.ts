const Layout = () => import("@/layout/index.vue");

export default {
  path: "/ops",
  name: "OpsManagement",
  component: Layout,
  redirect: "/ops/runtime",
  meta: {
    icon: "ri:tools-line",
    title: "运维管理",
    rank: 2
  },
  children: [
    {
      path: "/ops/runtime",
      name: "OpsRuntime",
      component: () => import("@/views/cmdb/ops/runtime.vue"),
      meta: {
        title: "运行状态",
        auths: "ops:runtime:view"
      }
    },
    {
      path: "/ops/logs",
      name: "OpsLogs",
      component: () => import("@/views/ops/logs.vue"),
      meta: {
        title: "日志中心",
        auths: "ops:logs:view"
      }
    },
    {
      path: "/ops/tasks",
      name: "OpsTasks",
      component: () => import("@/views/ops/tasks.vue"),
      meta: {
        title: "任务中心",
        auths: "ops:task:view"
      }
    },
    {
      path: "/ops/backup",
      name: "OpsBackup",
      component: () => import("@/views/cmdb/ops/backup.vue"),
      meta: {
        title: "备份恢复",
        auths: "ops:backup:view"
      }
    },
    {
      path: "/ops/sql",
      name: "OpsSqlConsole",
      component: () => import("@/views/ops/sql.vue"),
      meta: {
        title: "SQL 控制台",
        auths: ["ops:sql:read", "ops:sql:execute"]
      }
    }
  ]
} satisfies RouteConfigsTable;
