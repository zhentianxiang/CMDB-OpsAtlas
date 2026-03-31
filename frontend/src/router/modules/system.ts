import { $t } from "@/plugins/i18n";
const Layout = () => import("@/layout/index.vue");

export default {
  path: "/system",
  name: "System",
  component: Layout,
  redirect: "/system/user/index",
  meta: {
    icon: "ri:settings-3-line",
    title: $t("menus.pureSysManagement"),
    rank: 10
  },
  children: [
    {
      path: "/system/user/index",
      name: "SystemUser",
      component: () => import("@/views/system/user/index.vue"),
      meta: {
        icon: "ri:admin-line",
        title: $t("menus.pureUser"),
        auths: "sys:user:list"
      }
    },
    {
      path: "/system/role/index",
      name: "SystemRole",
      component: () => import("@/views/system/role/index.vue"),
      meta: {
        icon: "ri:admin-fill",
        title: $t("menus.pureRole"),
        auths: "sys:role:list"
      }
    },
    {
      path: "/system/menu/index",
      name: "SystemMenu",
      component: () => import("@/views/system/menu/index.vue"),
      meta: {
        icon: "ri:menu-fill",
        title: $t("menus.pureSystemMenu"),
        auths: "sys:menu:list"
      }
    },
    {
      path: "/system/dept/index",
      name: "SystemDept",
      component: () => import("@/views/system/dept/index.vue"),
      meta: {
        icon: "ri:git-branch-line",
        title: $t("menus.pureDept"),
        auths: "sys:dept:list"
      }
    },
    {
      path: "/system/audit/index",
      name: "SystemAudit",
      component: () => import("@/views/system/audit/index.vue"),
      meta: {
        icon: "ri:history-line",
        title: "操作审计",
        auths: "sys:audit:list"
      }
    }
  ]
} satisfies RouteConfigsTable;
