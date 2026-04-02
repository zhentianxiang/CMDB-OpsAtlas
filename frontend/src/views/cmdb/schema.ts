import type { AppItem, Cluster, DependencyItem, DomainItem, Host, PortItem } from "@/api/cmdb";

export type ResourceKey = "clusters" | "hosts" | "apps" | "ports" | "domains" | "dependencies";

export type CmdbTab = ResourceKey | "host-detail" | "topology";

export type ResourceConfig = {
  key: ResourceKey;
  title: string;
  path: string;
  columns: string[];
  fields: Array<{
    key: string;
    label: string;
    type?: "text" | "number" | "checkbox" | "select";
    required?: boolean;
    nullable?: boolean;
    optionsFrom?: "clusters" | "hosts" | "apps";
    options?: Array<{
      label: string;
      value: string | number;
    }>;
  }>;
};

export type PortDraft = {
  port: number | null;
  protocol: string;
};

export type ResourcePermissionMap = Record<
  ResourceKey,
  { create: string; update: string; delete: string }
>;

export type CmdbListState = {
  clusters: Cluster[];
  hosts: Host[];
  apps: AppItem[];
  ports: PortItem[];
  domains: DomainItem[];
  dependencies: DependencyItem[];
};

export const configs: ResourceConfig[] = [
  {
    key: "clusters",
    title: "集群",
    path: "/api/v1/clusters",
    columns: ["ID", "name", "type", "env", "remark"],
    fields: [
      { key: "name", label: "名称", required: true },
      { key: "type", label: "类型" },
      { key: "env", label: "环境" },
      { key: "remark", label: "备注" }
    ]
  },
  {
    key: "hosts",
    title: "主机",
    path: "/api/v1/hosts",
    columns: ["ID", "name", "address", "cluster_id", "status", "remark"],
    fields: [
      { key: "name", label: "名称", required: true },
      { key: "public_ip", label: "公网IP" },
      { key: "private_ip", label: "内网IP" },
      {
        key: "cluster_id",
        label: "集群",
        type: "select",
        optionsFrom: "clusters",
        nullable: true
      },
      { key: "cpu", label: "CPU", type: "number" },
      { key: "memory", label: "内存", type: "number" },
      { key: "os", label: "操作系统" },
      { key: "status", label: "状态" },
      { key: "remark", label: "备注" }
    ]
  },
  {
    key: "apps",
    title: "应用",
    path: "/api/v1/apps",
    columns: ["ID", "name", "host_id", "type", "version", "deploy_type", "remark"],
    fields: [
      { key: "name", label: "名称", required: true },
      {
        key: "host_id",
        label: "主机",
        type: "select",
        optionsFrom: "hosts",
        required: true
      },
      {
        key: "type",
        label: "类型",
        type: "select",
        options: [
          { label: "WEB应用", value: "WEB应用" },
          { label: "API服务", value: "API服务" },
          { label: "后台服务", value: "后台服务" },
          { label: "缓存", value: "缓存" },
          { label: "关系型数据库", value: "关系型数据库" },
          { label: "非关系型数据库", value: "非关系型数据库" },
          { label: "消息队列", value: "消息队列" },
          { label: "搜索引擎", value: "搜索引擎" },
          { label: "日志存储", value: "日志存储" },
          { label: "时序数据库", value: "时序数据库" },
          { label: "对象存储", value: "对象存储" },
          { label: "任务调度", value: "任务调度" },
          { label: "网关", value: "网关" },
          { label: "代理", value: "代理" },
          { label: "监控", value: "监控" },
          { label: "认证服务", value: "认证服务" },
          { label: "配置中心", value: "配置中心" },
          { label: "注册中心", value: "注册中心" },
          { label: "其他", value: "其他" }
        ]
      },
      { key: "version", label: "版本" },
      {
        key: "deploy_type",
        label: "部署方式",
        type: "select",
        options: [
          { label: "物理机", value: "物理机" },
          { label: "虚拟机", value: "虚拟机" },
          { label: "Docker", value: "Docker" },
          { label: "Kubernetes", value: "Kubernetes" },
          { label: "Serverless", value: "Serverless" },
          { label: "二进制", value: "二进制" },
          { label: "其他", value: "其他" }
        ]
      },
      { key: "remark", label: "备注" }
    ]
  },
  {
    key: "ports",
    title: "端口",
    path: "/api/v1/ports",
    columns: ["ID", "app_id", "port", "protocol", "is_public", "remark"],
    fields: [
      {
        key: "host_id",
        label: "主机",
        type: "select",
        optionsFrom: "hosts",
        nullable: true
      },
      {
        key: "app_id",
        label: "应用",
        type: "select",
        optionsFrom: "apps",
        required: true
      },
      { key: "port", label: "端口", type: "number", required: true },
      {
        key: "protocol",
        label: "协议",
        type: "select",
        options: [
          { label: "TCP", value: "TCP" },
          { label: "UDP", value: "UDP" },
          { label: "HTTP", value: "HTTP" },
          { label: "HTTPS", value: "HTTPS" }
        ]
      },
      { key: "is_public", label: "公网开放", type: "checkbox" },
      { key: "remark", label: "备注" }
    ]
  },
  {
    key: "domains",
    title: "域名",
    path: "/api/v1/domains",
    columns: ["ID", "domain", "app_id", "host_id", "remark"],
    fields: [
      { key: "domain", label: "域名", required: true },
      { key: "host_id", label: "主机", type: "select", optionsFrom: "hosts", nullable: true },
      { key: "app_id", label: "应用", type: "select", optionsFrom: "apps", nullable: true },
      { key: "remark", label: "备注" }
    ]
  },
  {
    key: "dependencies",
    title: "依赖",
    path: "/api/v1/dependencies",
    columns: [
      "ID",
      "source_app_id",
      "target_app_id",
      "domain_id",
      "source_host_id",
      "target_host_id",
      "source_node",
      "target_node",
      "desc",
      "remark"
    ],
    fields: [
      {
        key: "source_host_id",
        label: "调用方主机",
        type: "select",
        optionsFrom: "hosts",
        nullable: true
      },
      {
        key: "source_app_id",
        label: "调用方应用",
        type: "select",
        optionsFrom: "apps",
        nullable: true
      },
      {
        key: "domain_id",
        label: "访问域名",
        type: "select",
        nullable: true
      },
      {
        key: "target_host_id",
        label: "被调用主机",
        type: "select",
        optionsFrom: "hosts",
        nullable: true
      },
      {
        key: "target_app_id",
        label: "被调用应用",
        type: "select",
        optionsFrom: "apps",
        nullable: true
      },
      { key: "source_node", label: "调用方外部节点", nullable: true },
      { key: "target_node", label: "被调用外部节点", nullable: true },
      { key: "desc", label: "描述" },
      { key: "remark", label: "备注" }
    ]
  }
];

export const resourcePermissionMap: ResourcePermissionMap = {
  clusters: {
    create: "cmdb:cluster:create",
    update: "cmdb:cluster:update",
    delete: "cmdb:cluster:delete"
  },
  hosts: {
    create: "cmdb:host:create",
    update: "cmdb:host:update",
    delete: "cmdb:host:delete"
  },
  apps: {
    create: "cmdb:app:create",
    update: "cmdb:app:update",
    delete: "cmdb:app:delete"
  },
  ports: {
    create: "cmdb:port:create",
    update: "cmdb:port:update",
    delete: "cmdb:port:delete"
  },
  domains: {
    create: "cmdb:domain:create",
    update: "cmdb:domain:update",
    delete: "cmdb:domain:delete"
  },
  dependencies: {
    create: "cmdb:dependency:create",
    update: "cmdb:dependency:update",
    delete: "cmdb:dependency:delete"
  }
};

export const searchPlaceholders: Partial<Record<ResourceKey, string>> = {
  clusters: "搜索集群名称 / 类型 / 环境",
  hosts: "搜索主机名称 / 地址 / 集群 / 状态",
  apps: "搜索应用名称 / 类型 / 版本 / 主机",
  ports: "搜索端口 / 协议 / 应用",
  domains: "搜索域名 / 应用 / 主机",
  dependencies: "搜索源目标应用 / 主机 / 外部节点 / 描述"
};
