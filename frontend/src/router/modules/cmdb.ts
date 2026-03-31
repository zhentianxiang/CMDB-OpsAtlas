import { $t } from "@/plugins/i18n";
const Layout = () => import("@/layout/index.vue");

export default {
  path: "/cmdb",
  name: "CMDB",
  component: Layout,
  redirect: "/cmdb/resources",
  meta: {
    icon: "ri:database-2-line",
    title: $t("menus.pureCMDB"),
    rank: 1,
    auths: "cmdb:view"
  },
  children: [
    {
      path: "/cmdb/resources",
      name: "CMDBResources",
      component: () => import("@/views/cmdb/index.vue"),
      props: {
        defaultTab: "clusters"
      },
      meta: {
        title: "CMDB 资源",
        auths: "cmdb:view"
      }
    }
  ]
} satisfies RouteConfigsTable;
