<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue";
import type { AppItem, Cluster, DependencyItem, DomainItem, Host, PortItem } from "@/api/cmdb";

type LaneKey = "external" | "host" | "entry" | "service" | "data";
type ActiveLane = "all" | LaneKey;

type NodePayload =
  | {
      nodeType: "host";
      host: Host;
      clusterName: string;
      appsText: string;
      portsText: string;
    }
  | {
      nodeType: "app";
      app: AppItem;
      hostName: string;
      ports: string[];
      domains: string[];
      inCount: number;
      outCount: number;
      laneName: string;
    }
  | {
      nodeType: "external";
      name: string;
      laneName: string;
      sourceHint: string;
    };

type GraphNode = {
  id: string;
  laneKey: LaneKey;
  x: number;
  y: number;
  width: number;
  height: number;
  headerColor: string;
  title: string;
  bodyLines: string[];
  dataPayload: NodePayload;
};

type GraphEdge = {
  id: string;
  source: string;
  target: string;
  label: string;
  color: string;
};

type GraphBounds = {
  minX: number;
  minY: number;
  maxX: number;
  maxY: number;
  width: number;
  height: number;
};

type LaneMeta = {
  key: LaneKey;
  label: string;
  color: string;
  bg: string;
  x: number;
  width: number;
};

const props = defineProps<{
  hosts: Host[];
  apps: AppItem[];
  dependencies: DependencyItem[];
  ports: PortItem[];
  domains: DomainItem[];
  clusters: Cluster[];
  selectedClusterId: number | null;
  selectedHostId: number | null;
  selectedAppId: number | null;
  selectedDomainId: number | null;
}>();

const emit = defineEmits<{
  (e: "node-click", payload: NodePayload): void;
}>();

const wrapRef = ref<HTMLDivElement | null>(null);
const viewportRef = ref<HTMLDivElement | null>(null);
const fullscreenMode = ref(false);
const hasGraphData = ref(true);
const activeLane = ref<ActiveLane>("all");
const selectedNodeId = ref<string | null>(null);
const sceneScale = ref(1);
const scenePan = reactive({ x: 0, y: 0 });
const skipClick = ref(false);
const isPanning = ref(false);

const laneLegends = [
  { key: "external", label: "外部", color: "#7c3aed", bg: "rgba(124, 58, 237, 0.16)" },
  { key: "host", label: "主机层", color: "#2563eb", bg: "rgba(37, 99, 235, 0.16)" },
  { key: "entry", label: "应用入口层", color: "#0891b2", bg: "rgba(8, 145, 178, 0.16)" },
  { key: "service", label: "服务层", color: "#f97316", bg: "rgba(249, 115, 22, 0.16)" },
  { key: "data", label: "数据层", color: "#e11d48", bg: "rgba(225, 29, 72, 0.16)" }
] as const;

const graphState = ref<{
  nodes: GraphNode[];
  edges: GraphEdge[];
  lanes: LaneMeta[];
  bounds: GraphBounds;
}>({
  nodes: [],
  edges: [],
  lanes: [],
  bounds: { minX: 0, minY: 0, maxX: 1200, maxY: 760, width: 1200, height: 760 }
});

const nodePositions = ref<Record<string, { x: number; y: number }>>({});

const interaction = ref<
  | null
  | {
      type: "pan";
      pointerId: number;
      startX: number;
      startY: number;
      originPanX: number;
      originPanY: number;
    }
  | {
      type: "node";
      pointerId: number;
      nodeId: string;
      offsetX: number;
      offsetY: number;
    }
>(null);

function getEntityId(item: any) {
  const raw = item?.ID ?? item?.id ?? null;
  const id = Number(raw);
  return Number.isFinite(id) && id > 0 ? id : null;
}

function getHostIdFromApp(item: any) {
  const raw = item?.host_id ?? item?.hostId ?? null;
  const id = Number(raw);
  return Number.isFinite(id) && id > 0 ? id : null;
}

function getClusterIdFromHost(item: any) {
  const raw = item?.cluster_id ?? item?.clusterId ?? null;
  if (raw === null || raw === undefined || raw === "") return null;
  const id = Number(raw);
  return Number.isFinite(id) && id > 0 ? id : null;
}

const hostMap = computed(() => {
  const map = new Map<number, Host>();
  props.hosts.forEach(item => {
    const id = getEntityId(item);
    if (!id) return;
    map.set(id, item);
  });
  return map;
});

const appMap = computed(() => {
  const map = new Map<number, AppItem>();
  props.apps.forEach(item => {
    const id = getEntityId(item);
    if (!id) return;
    map.set(id, item);
  });
  return map;
});

const clusterMap = computed(() => {
  const map = new Map<number, Cluster>();
  props.clusters.forEach(item => {
    const id = getEntityId(item);
    if (!id) return;
    map.set(id, item);
  });
  return map;
});

const portsByApp = computed(() => {
  const map = new Map<number, PortItem[]>();
  props.ports.forEach(item => {
    const current = map.get(item.app_id) || [];
    current.push(item);
    map.set(item.app_id, current);
  });
  return map;
});

const domainsByApp = computed(() => {
  const map = new Map<number, DomainItem[]>();
  props.domains.forEach(item => {
    if (!item.app_id) return;
    const current = map.get(item.app_id) || [];
    current.push(item);
    map.set(item.app_id, current);
  });
  return map;
});

const appInOutCount = computed(() => {
  const map = new Map<number, { out: number; in: number }>();
  props.dependencies.forEach(dep => {
    if (dep.source_app_id) {
      const source = map.get(dep.source_app_id) || { out: 0, in: 0 };
      source.out += 1;
      map.set(dep.source_app_id, source);
    }
    if (dep.target_app_id) {
      const target = map.get(dep.target_app_id) || { out: 0, in: 0 };
      target.in += 1;
      map.set(dep.target_app_id, target);
    }
  });
  return map;
});

function classifyAppLane(app: AppItem): LaneKey {
  const t = (app.type || "").toLowerCase();
  if (t.includes("web") || t.includes("nginx") || t.includes("gateway")) return "entry";
  if (t.includes("db") || t.includes("data") || t.includes("mysql") || t.includes("pgsql")) return "data";
  if (t.includes("cache") || t.includes("redis") || t.includes("mq")) return "data";
  return "service";
}

function buildGraphData() {
  const clusterHosts = props.hosts.filter(item => {
    if (!props.selectedClusterId) return true;
    return getClusterIdFromHost(item) === props.selectedClusterId;
  });
  const clusterHostIds = new Set(clusterHosts.map(item => getEntityId(item)).filter(Boolean) as number[]);
  const clusterApps = props.apps.filter(item => {
    const hostId = getHostIdFromApp(item);
    return !!hostId && clusterHostIds.has(hostId);
  });
  const clusterAppIds = new Set(clusterApps.map(item => getEntityId(item)).filter(Boolean) as number[]);

  const selectedDomain = props.selectedDomainId
    ? props.domains.find(item => getEntityId(item) === props.selectedDomainId) || null
    : null;
  const strictFocusAppId = props.selectedAppId && clusterAppIds.has(props.selectedAppId) ? props.selectedAppId : null;
  const strictFocusApp = strictFocusAppId
    ? clusterApps.find(item => getEntityId(item) === strictFocusAppId) || null
    : null;
  const strictFocusHostId = strictFocusApp ? getHostIdFromApp(strictFocusApp) : null;

  const seedHostIds = new Set<number>();
  const seedAppIds = new Set<number>();

  if (props.selectedHostId && clusterHostIds.has(props.selectedHostId)) {
    seedHostIds.add(props.selectedHostId);
  }
  if (props.selectedAppId && clusterAppIds.has(props.selectedAppId)) {
    seedAppIds.add(props.selectedAppId);
  }
  if (selectedDomain) {
    if (selectedDomain.app_id && clusterAppIds.has(selectedDomain.app_id)) {
      seedAppIds.add(selectedDomain.app_id);
    }
    if (selectedDomain.host_id && clusterHostIds.has(selectedDomain.host_id)) {
      seedHostIds.add(selectedDomain.host_id);
    }
  }

  if (seedHostIds.size > 0) {
    clusterApps.forEach(item => {
      const hostId = getHostIdFromApp(item);
      const appId = getEntityId(item);
      if (!hostId || !appId) return;
      if (seedHostIds.has(hostId)) seedAppIds.add(appId);
    });
  }

  const hasStrictAppFocus = !!strictFocusAppId && seedAppIds.has(strictFocusAppId);
  const useFocusedFilter = seedHostIds.size > 0 || seedAppIds.size > 0;
  const selectedHostIds = new Set<number>();
  const selectedAppIds = new Set<number>();

  if (hasStrictAppFocus) {
    selectedAppIds.add(strictFocusAppId!);

    if (strictFocusHostId && clusterHostIds.has(strictFocusHostId)) {
      selectedHostIds.add(strictFocusHostId);
    }

    props.dependencies.forEach(dep => {
      const related = dep.source_app_id === strictFocusAppId || dep.target_app_id === strictFocusAppId;

      if (!related) return;

      if (dep.source_app_id && clusterAppIds.has(dep.source_app_id)) {
        selectedAppIds.add(dep.source_app_id);
      }
      if (dep.target_app_id && clusterAppIds.has(dep.target_app_id)) {
        selectedAppIds.add(dep.target_app_id);
      }
      if (dep.source_host_id && clusterHostIds.has(dep.source_host_id)) {
        selectedHostIds.add(dep.source_host_id);
      }
      if (dep.target_host_id && clusterHostIds.has(dep.target_host_id)) {
        selectedHostIds.add(dep.target_host_id);
      }
    });
  } else if (useFocusedFilter) {
    const outgoingByApp = new Map<number, Array<{ appId: number | null; hostId: number | null }>>();
    const outgoingByHost = new Map<number, Array<{ appId: number | null; hostId: number | null }>>();

    props.dependencies.forEach(dep => {
      if (dep.source_app_id) {
        const current = outgoingByApp.get(dep.source_app_id) || [];
        current.push({ appId: dep.target_app_id || null, hostId: dep.target_host_id || null });
        outgoingByApp.set(dep.source_app_id, current);
      }
      if (dep.source_host_id) {
        const current = outgoingByHost.get(dep.source_host_id) || [];
        current.push({ appId: dep.target_app_id || null, hostId: dep.target_host_id || null });
        outgoingByHost.set(dep.source_host_id, current);
      }
    });

    const appQueue = [...seedAppIds];
    const hostQueue = [...seedHostIds];

    appQueue.forEach(id => selectedAppIds.add(id));
    hostQueue.forEach(id => selectedHostIds.add(id));

    while (appQueue.length > 0) {
      const current = appQueue.shift()!;
      (outgoingByApp.get(current) || []).forEach(next => {
        if (next.appId && clusterAppIds.has(next.appId) && !selectedAppIds.has(next.appId)) {
          selectedAppIds.add(next.appId);
          appQueue.push(next.appId);
        }
        if (next.hostId && clusterHostIds.has(next.hostId) && !selectedHostIds.has(next.hostId)) {
          selectedHostIds.add(next.hostId);
          hostQueue.push(next.hostId);
        }
      });
    }

    while (hostQueue.length > 0) {
      const current = hostQueue.shift()!;
      clusterApps.forEach(item => {
        const hostId = getHostIdFromApp(item);
        const appId = getEntityId(item);
        if (!hostId || !appId || hostId !== current || selectedAppIds.has(appId)) return;
        selectedAppIds.add(appId);
        appQueue.push(appId);
      });
      (outgoingByHost.get(current) || []).forEach(next => {
        if (next.appId && clusterAppIds.has(next.appId) && !selectedAppIds.has(next.appId)) {
          selectedAppIds.add(next.appId);
          appQueue.push(next.appId);
        }
        if (next.hostId && clusterHostIds.has(next.hostId) && !selectedHostIds.has(next.hostId)) {
          selectedHostIds.add(next.hostId);
          hostQueue.push(next.hostId);
        }
      });
    }
  } else {
    clusterHostIds.forEach(id => selectedHostIds.add(id));
    clusterAppIds.forEach(id => selectedAppIds.add(id));
  }

  const selectedHosts = clusterHosts.filter(item => {
    const id = getEntityId(item);
    return !!id && selectedHostIds.has(id);
  });
  const selectedApps = clusterApps.filter(item => {
    const id = getEntityId(item);
    return !!id && selectedAppIds.has(id);
  });

  const graphWidth = Math.max((viewportRef.value?.clientWidth || 1360) - 80, 980);
  const laneWidth = Math.max(graphWidth / 5, 220);
  const laneDefinitions = laneLegends.map((lane, index) => ({
    ...lane,
    x: 80 + laneWidth * index + laneWidth / 2,
    width: laneWidth - 18
  })) as LaneMeta[];
  const laneX = new Map<LaneKey, number>(laneDefinitions.map(item => [item.key as LaneKey, item.x]));

  const nodes: GraphNode[] = [];
  const edges: GraphEdge[] = [];
  const externalNodes = new Map<string, GraphNode>();
  const laneCursor: Record<LaneKey, number> = {
    external: 146,
    host: 146,
    entry: 146,
    service: 146,
    data: 146
  };
  const appPosition = new Map<number, { x: number; y: number; lane: LaneKey }>();

  selectedHosts.forEach((host, index) => {
    const hostY = 160 + index * 206;
    const hostId = getEntityId(host);
    if (!hostId) return;
    const clusterId = getClusterIdFromHost(host);
    const cluster = clusterId ? clusterMap.value.get(clusterId) : null;
    const hostApps = selectedApps.filter(item => getHostIdFromApp(item) === hostId);

    nodes.push({
      id: `host-${hostId}`,
      laneKey: "host",
      x: laneX.get("host") || 340,
      y: hostY,
      width: 320,
      height: 170,
      headerColor: "#2563eb",
      title: host.name,
      bodyLines: [
        "层级: 主机层",
        `IP: ${host.private_ip || host.public_ip || host.ip || "-"}`,
        `状态: ${host.status || "-"}`,
        `集群: ${cluster?.name || "未归属"}`
      ],
      dataPayload: {
        nodeType: "host",
        host,
        clusterName: cluster?.name || "未归属",
        appsText: hostApps.map(item => item.name).join("、") || "-",
        portsText:
          hostApps
            .flatMap(app =>
              (portsByApp.value.get(getEntityId(app) || 0) || []).map(p => `${app.name}:${p.port}/${p.protocol || "TCP"}`)
            )
            .join("\n") || "-"
      }
    });

    hostApps.forEach((app, appIndex) => {
      const appId = getEntityId(app);
      if (!appId) return;
      const lane = classifyAppLane(app);
      const preferredY = hostY + (appIndex - (hostApps.length - 1) / 2) * 132;
      const y = Math.max(preferredY, laneCursor[lane]);
      laneCursor[lane] = y + 142;
      appPosition.set(appId, {
        x: laneX.get(lane) || 860,
        y,
        lane
      });
    });
  });

  selectedApps.forEach(app => {
    const appId = getEntityId(app);
    if (!appId) return;
    const hostId = getHostIdFromApp(app);
    const pos = appPosition.get(appId);
    const lane = pos?.lane || "service";
    const ports = (portsByApp.value.get(appId) || []).map(item => `${item.port}/${item.protocol || "TCP"}`);
    const io = appInOutCount.value.get(appId) || { in: 0, out: 0 };
    const laneName = lane === "entry" ? "应用入口层" : lane === "data" ? "数据层" : "服务层";
    nodes.push({
      id: `app-${appId}`,
      laneKey: lane,
      x: pos?.x || (laneX.get("service") || 960),
      y: pos?.y || laneCursor.service + 20,
      width: 320,
      height: 188,
      headerColor: lane === "entry" ? "#0891b2" : lane === "data" ? "#e11d48" : "#f97316",
      title: app.name,
      bodyLines: [
        `层级: ${laneName}`,
        `主机: ${hostMap.value.get(hostId || 0)?.name || "-"}`,
        `端口: ${(ports || []).slice(0, 3).join(", ") || "-"}`,
        `类型: ${app.type || "-"} | 调用: ↑${io.out} ↓${io.in}`
      ],
      dataPayload: {
        nodeType: "app",
        app,
        hostName: hostMap.value.get(hostId || 0)?.name || "-",
        ports,
        domains: (domainsByApp.value.get(appId) || []).map(item => item.domain),
        inCount: io.in,
        outCount: io.out,
        laneName
      }
    });
  });

  selectedApps.forEach(app => {
    const appId = getEntityId(app);
    const hostId = getHostIdFromApp(app);
    if (!appId || !hostId) return;
    edges.push({
      id: `deploy-${hostId}-${appId}`,
      source: `host-${hostId}`,
      target: `app-${appId}`,
      label: "部署",
      color: "#2563eb"
    });
  });

  function ensureExternalNode(id: string, display: string) {
    if (externalNodes.has(id)) return;
    const y = laneCursor.external;
    laneCursor.external += 142;
    externalNodes.set(id, {
      id,
      laneKey: "external",
      x: laneX.get("external") || 120,
      y,
      width: 286,
      height: 136,
      headerColor: "#7c3aed",
      title: display,
      bodyLines: ["层级: 外部", "来源: 未登记节点"],
      dataPayload: {
        nodeType: "external",
        name: display,
        laneName: "外部",
        sourceHint: "来自 source_node/target_node 或未入图实体"
      }
    });
  }

  function normalizeExternalKey(text: string) {
    return text
      .trim()
      .replace(/\s+/g, "_")
      .replace(/[^\w\-:.]/g, "_")
      .slice(0, 80);
  }

  function endpointNodeId(appId: number | null, hostId: number | null, nodeText: string | null) {
    if (appId) {
      if (selectedAppIds.has(appId)) return `app-${appId}`;
      const name = appMap.value.get(appId)?.name || `App-${appId}`;
      const extId = `external-app-${appId}`;
      ensureExternalNode(extId, `[外部应用] ${name}`);
      return extId;
    }
    if (hostId) {
      if (selectedHostIds.has(hostId)) return `host-${hostId}`;
      const name = hostMap.value.get(hostId)?.name || `Host-${hostId}`;
      const extId = `external-host-${hostId}`;
      ensureExternalNode(extId, `[外部主机] ${name}`);
      return extId;
    }
    if (nodeText && nodeText.trim()) {
      const text = nodeText.trim();
      const extId = `external-node-${normalizeExternalKey(text)}`;
      ensureExternalNode(extId, `[外部节点] ${text}`);
      return extId;
    }
    return null;
  }

  props.dependencies.forEach(dep => {
    if (hasStrictAppFocus) {
      const related = dep.source_app_id === strictFocusAppId || dep.target_app_id === strictFocusAppId;
      if (!related) return;
    }

    const source = endpointNodeId(dep.source_app_id, dep.source_host_id, dep.source_node);
    const target = endpointNodeId(dep.target_app_id, dep.target_host_id, dep.target_node);
    if (!source || !target || source === target) return;
    const depId = dep.ID ?? `${source}->${target}->${dep.desc || "dep"}`;
    edges.push({
      id: `dep-${depId}`,
      source,
      target,
      label: dep.desc || "调用",
      color: "#e11d48"
    });
  });

  nodes.push(...externalNodes.values());

  const nodeIdSet = new Set(nodes.map(item => item.id));
  const validEdges = edges.filter(item => nodeIdSet.has(item.source) && nodeIdSet.has(item.target) && item.source !== item.target);

  const bounds = nodes.reduce<GraphBounds>(
    (acc, node) => ({
      minX: Math.min(acc.minX, node.x - node.width / 2),
      minY: Math.min(acc.minY, node.y - node.height / 2),
      maxX: Math.max(acc.maxX, node.x + node.width / 2),
      maxY: Math.max(acc.maxY, node.y + node.height / 2),
      width: 0,
      height: 0
    }),
    { minX: Number.POSITIVE_INFINITY, minY: Number.POSITIVE_INFINITY, maxX: 0, maxY: 0, width: 0, height: 0 }
  );

  if (!nodes.length) {
    return {
      nodes,
      edges: validEdges,
      lanes: laneDefinitions,
      bounds: { minX: 0, minY: 0, maxX: graphWidth, maxY: 760, width: graphWidth, height: 760 }
    };
  }

  bounds.minX -= 120;
  bounds.minY = Math.max(0, bounds.minY - 80);
  bounds.maxX += 140;
  bounds.maxY += 120;
  bounds.width = bounds.maxX - bounds.minX;
  bounds.height = bounds.maxY - bounds.minY;

  return { nodes, edges: validEdges, lanes: laneDefinitions, bounds };
}

function renderGraph() {
  const data = buildGraphData();
  graphState.value = data;
  nodePositions.value = Object.fromEntries(data.nodes.map(node => [node.id, { x: node.x, y: node.y }]));
  hasGraphData.value = data.nodes.length > 0;
  if (selectedNodeId.value && !data.nodes.some(node => node.id === selectedNodeId.value)) {
    selectedNodeId.value = null;
  }
}

const renderedNodes = computed(() =>
  graphState.value.nodes.map(node => {
    const override = nodePositions.value[node.id];
    return {
      ...node,
      x: override?.x ?? node.x,
      y: override?.y ?? node.y
    };
  })
);

const renderedNodeMap = computed(() => new Map(renderedNodes.value.map(node => [node.id, node])));

const chainNodeIds = computed(() => {
  if (!selectedNodeId.value) return new Set<string>();
  const forward = new Map<string, string[]>();
  graphState.value.edges.forEach(edge => {
    const current = forward.get(edge.source) || [];
    current.push(edge.target);
    forward.set(edge.source, current);
  });
  const visited = new Set<string>([selectedNodeId.value]);
  const queue = [selectedNodeId.value];
  while (queue.length > 0) {
    const current = queue.shift()!;
    (forward.get(current) || []).forEach(target => {
      if (visited.has(target)) return;
      visited.add(target);
      queue.push(target);
    });
  }
  return visited;
});

function isNodeDim(node: GraphNode) {
  if (selectedNodeId.value) return !chainNodeIds.value.has(node.id);
  if (activeLane.value === "all") return false;
  return node.laneKey !== activeLane.value;
}

function isNodeFocus(node: GraphNode) {
  if (selectedNodeId.value) return chainNodeIds.value.has(node.id);
  if (activeLane.value === "all") return false;
  return node.laneKey === activeLane.value;
}

function getAnchor(node: GraphNode, side: "left" | "right" | "top" | "bottom") {
  if (side === "left") return { x: node.x - node.width / 2, y: node.y };
  if (side === "right") return { x: node.x + node.width / 2, y: node.y };
  if (side === "top") return { x: node.x, y: node.y - node.height / 2 };
  return { x: node.x, y: node.y + node.height / 2 };
}

function getEdgeGeometry(source: GraphNode, target: GraphNode) {
  const dx = target.x - source.x;
  const dy = target.y - source.y;
  const horizontal = Math.abs(dx) >= Math.abs(dy);
  const start = horizontal ? getAnchor(source, dx >= 0 ? "right" : "left") : getAnchor(source, dy >= 0 ? "bottom" : "top");
  const end = horizontal ? getAnchor(target, dx >= 0 ? "left" : "right") : getAnchor(target, dy >= 0 ? "top" : "bottom");
  const control1 = horizontal
    ? { x: start.x + dx * 0.38, y: start.y }
    : { x: start.x, y: start.y + dy * 0.38 };
  const control2 = horizontal
    ? { x: end.x - dx * 0.38, y: end.y }
    : { x: end.x, y: end.y - dy * 0.38 };
  const path = `M ${start.x} ${start.y} C ${control1.x} ${control1.y}, ${control2.x} ${control2.y}, ${end.x} ${end.y}`;
  const t = 0.5;
  const x =
    Math.pow(1 - t, 3) * start.x +
    3 * Math.pow(1 - t, 2) * t * control1.x +
    3 * (1 - t) * Math.pow(t, 2) * control2.x +
    Math.pow(t, 3) * end.x;
  const y =
    Math.pow(1 - t, 3) * start.y +
    3 * Math.pow(1 - t, 2) * t * control1.y +
    3 * (1 - t) * Math.pow(t, 2) * control2.y +
    Math.pow(t, 3) * end.y;
  return { path, labelX: x, labelY: y };
}

const renderedEdges = computed(() =>
  graphState.value.edges
    .map(edge => {
      const source = renderedNodeMap.value.get(edge.source);
      const target = renderedNodeMap.value.get(edge.target);
      if (!source || !target) return null;
      const geometry = getEdgeGeometry(source, target);
      const active = selectedNodeId.value
        ? chainNodeIds.value.has(edge.source) && chainNodeIds.value.has(edge.target)
        : activeLane.value === "all"
          ? false
          : source.laneKey === activeLane.value || target.laneKey === activeLane.value;
      const dim = selectedNodeId.value || activeLane.value !== "all" ? !active : false;
      return {
        ...edge,
        ...geometry,
        focus: active,
        dim
      };
    })
    .filter(Boolean) as Array<GraphEdge & { path: string; labelX: number; labelY: number; focus: boolean; dim: boolean }>
);

const sceneBounds = computed(() => {
  if (!renderedNodes.value.length) return graphState.value.bounds;
  const bounds = renderedNodes.value.reduce<GraphBounds>(
    (acc, node) => ({
      minX: Math.min(acc.minX, node.x - node.width / 2),
      minY: Math.min(acc.minY, node.y - node.height / 2),
      maxX: Math.max(acc.maxX, node.x + node.width / 2),
      maxY: Math.max(acc.maxY, node.y + node.height / 2),
      width: 0,
      height: 0
    }),
    { minX: Number.POSITIVE_INFINITY, minY: Number.POSITIVE_INFINITY, maxX: 0, maxY: 0, width: 0, height: 0 }
  );
  bounds.minX -= 120;
  bounds.minY = Math.max(0, bounds.minY - 80);
  bounds.maxX += 140;
  bounds.maxY += 120;
  bounds.width = bounds.maxX - bounds.minX;
  bounds.height = bounds.maxY - bounds.minY;
  return bounds;
});

const sceneStyle = computed(() => ({
  width: `${Math.max(sceneBounds.value.maxX + 40, 1200)}px`,
  height: `${Math.max(sceneBounds.value.maxY + 40, 760)}px`,
  transform: `translate(${scenePan.x}px, ${scenePan.y}px) scale(${sceneScale.value})`
}));

function resetView() {
  const viewport = viewportRef.value;
  if (!viewport) return;
  const bounds = sceneBounds.value;
  const width = viewport.clientWidth || 1;
  const height = viewport.clientHeight || 1;
  const nextScale = Math.max(0.38, Math.min(1.05, Math.min((width - 72) / bounds.width, (height - 72) / bounds.height)));
  sceneScale.value = nextScale;
  scenePan.x = (width - bounds.width * nextScale) / 2 - bounds.minX * nextScale;
  scenePan.y = (height - bounds.height * nextScale) / 2 - bounds.minY * nextScale;
}

function toggleLane(lane: LaneKey) {
  activeLane.value = activeLane.value === lane ? "all" : lane;
  selectedNodeId.value = null;
}

function toScenePoint(clientX: number, clientY: number) {
  const rect = viewportRef.value?.getBoundingClientRect();
  if (!rect) return { x: 0, y: 0 };
  return {
    x: (clientX - rect.left - scenePan.x) / sceneScale.value,
    y: (clientY - rect.top - scenePan.y) / sceneScale.value
  };
}

function finishInteraction(moved: boolean) {
  if (moved) {
    skipClick.value = true;
    window.setTimeout(() => {
      skipClick.value = false;
    }, 0);
  }
  interaction.value = null;
  isPanning.value = false;
  window.removeEventListener("pointermove", onGlobalPointerMove);
  window.removeEventListener("pointerup", onGlobalPointerUp);
  window.removeEventListener("pointercancel", onGlobalPointerUp);
}

function onViewportPointerDown(event: PointerEvent) {
  if (event.button !== 0) return;
  const target = event.target as HTMLElement;
  if (target.closest(".topology-node")) return;
  interaction.value = {
    type: "pan",
    pointerId: event.pointerId,
    startX: event.clientX,
    startY: event.clientY,
    originPanX: scenePan.x,
    originPanY: scenePan.y
  };
  isPanning.value = true;
  window.addEventListener("pointermove", onGlobalPointerMove);
  window.addEventListener("pointerup", onGlobalPointerUp);
  window.addEventListener("pointercancel", onGlobalPointerUp);
}

function onNodePointerDown(event: PointerEvent, node: GraphNode) {
  if (event.button !== 0) return;
  const point = toScenePoint(event.clientX, event.clientY);
  interaction.value = {
    type: "node",
    pointerId: event.pointerId,
    nodeId: node.id,
    offsetX: point.x - node.x,
    offsetY: point.y - node.y
  };
  window.addEventListener("pointermove", onGlobalPointerMove);
  window.addEventListener("pointerup", onGlobalPointerUp);
  window.addEventListener("pointercancel", onGlobalPointerUp);
}

function onGlobalPointerMove(event: PointerEvent) {
  const current = interaction.value;
  if (!current || current.pointerId !== event.pointerId) return;
  if (current.type === "pan") {
    const dx = event.clientX - current.startX;
    const dy = event.clientY - current.startY;
    scenePan.x = current.originPanX + dx;
    scenePan.y = current.originPanY + dy;
    if (Math.abs(dx) + Math.abs(dy) > 4) skipClick.value = true;
    return;
  }
  const point = toScenePoint(event.clientX, event.clientY);
  nodePositions.value = {
    ...nodePositions.value,
    [current.nodeId]: {
      x: point.x - current.offsetX,
      y: point.y - current.offsetY
    }
  };
  skipClick.value = true;
}

function onGlobalPointerUp(event: PointerEvent) {
  const current = interaction.value;
  if (!current || current.pointerId !== event.pointerId) return;
  finishInteraction(skipClick.value);
}

function onCanvasClick() {
  if (skipClick.value) return;
  selectedNodeId.value = null;
}

function onNodeClick(node: GraphNode) {
  if (skipClick.value) return;
  selectedNodeId.value = node.id;
  emit("node-click", node.dataPayload);
}

function onWheel(event: WheelEvent) {
  const viewport = viewportRef.value;
  if (!viewport) return;
  const rect = viewport.getBoundingClientRect();
  const px = event.clientX - rect.left;
  const py = event.clientY - rect.top;
  const worldX = (px - scenePan.x) / sceneScale.value;
  const worldY = (py - scenePan.y) / sceneScale.value;
  const factor = event.deltaY < 0 ? 1.1 : 0.9;
  const nextScale = Math.min(2.2, Math.max(0.3, Number((sceneScale.value * factor).toFixed(3))));
  sceneScale.value = nextScale;
  scenePan.x = px - worldX * nextScale;
  scenePan.y = py - worldY * nextScale;
}

function zoomIn() {
  sceneScale.value = Math.min(2.2, Number((sceneScale.value + 0.14).toFixed(2)));
}

function zoomOut() {
  sceneScale.value = Math.max(0.3, Number((sceneScale.value - 0.14).toFixed(2)));
}

async function toggleFullscreen() {
  if (!wrapRef.value) return;
  if (document.fullscreenElement) await document.exitFullscreen();
  else await wrapRef.value.requestFullscreen();
  fullscreenMode.value = Boolean(document.fullscreenElement);
  window.setTimeout(() => {
    resetView();
  }, 80);
}

const onFullscreenChange = () => {
  fullscreenMode.value = Boolean(document.fullscreenElement);
  window.setTimeout(() => {
    resetView();
  }, 80);
};

watch(
  () => [
    props.selectedClusterId,
    props.selectedHostId,
    props.selectedAppId,
    props.selectedDomainId,
    props.hosts,
    props.apps,
    props.dependencies,
    props.ports,
    props.domains
  ],
  async () => {
    renderGraph();
    await nextTick();
    resetView();
  },
  { deep: true }
);

onMounted(async () => {
  document.addEventListener("fullscreenchange", onFullscreenChange);
  window.addEventListener("resize", resetView);
  renderGraph();
  await nextTick();
  resetView();
});

onBeforeUnmount(() => {
  document.removeEventListener("fullscreenchange", onFullscreenChange);
  window.removeEventListener("resize", resetView);
  window.removeEventListener("pointermove", onGlobalPointerMove);
  window.removeEventListener("pointerup", onGlobalPointerUp);
  window.removeEventListener("pointercancel", onGlobalPointerUp);
});
</script>

<template>
  <div ref="wrapRef" class="topology-shell">
    <div class="topology-toolbar">
      <div class="lane-legend" aria-hidden="true">
        <button
          v-for="legend in laneLegends"
          :key="legend.key"
          class="lane-chip"
          :class="{ active: activeLane === legend.key }"
          :style="{ color: legend.color, backgroundColor: legend.bg, borderColor: `${legend.color}44` }"
          @click="toggleLane(legend.key as LaneKey)"
        >
          {{ legend.label }}
        </button>
      </div>
      <div class="topology-actions">
        <el-button class="topology-action-btn" circle :title="fullscreenMode ? '退出全屏' : '全屏'" @click="toggleFullscreen">
          {{ fullscreenMode ? "⤫" : "⛶" }}
        </el-button>
        <el-button class="topology-action-btn" circle title="放大" @click="zoomIn">+</el-button>
        <el-button class="topology-action-btn" circle title="缩小" @click="zoomOut">-</el-button>
        <el-button class="topology-action-btn" circle title="适配视图" @click="resetView">↺</el-button>
      </div>
    </div>

    <div
      ref="viewportRef"
      class="topology-viewport"
      :class="{ panning: isPanning }"
      @wheel.prevent="onWheel"
      @pointerdown="onViewportPointerDown"
      @click="onCanvasClick"
    >
      <div class="topology-scene" :style="sceneStyle">
        <svg
          class="edge-layer"
          :width="Math.max(sceneBounds.maxX + 40, 1200)"
          :height="Math.max(sceneBounds.maxY + 40, 760)"
          :viewBox="`0 0 ${Math.max(sceneBounds.maxX + 40, 1200)} ${Math.max(sceneBounds.maxY + 40, 760)}`"
        >
          <defs>
            <marker id="arrow-blue" markerWidth="10" markerHeight="10" refX="8" refY="5" orient="auto">
              <path d="M 0 0 L 10 5 L 0 10 z" fill="#2563eb" />
            </marker>
            <marker id="arrow-orange" markerWidth="10" markerHeight="10" refX="8" refY="5" orient="auto">
              <path d="M 0 0 L 10 5 L 0 10 z" fill="#e11d48" />
            </marker>
          </defs>

          <g v-for="edge in renderedEdges" :key="edge.id" class="edge-group" :class="{ dim: edge.dim, focus: edge.focus }">
            <path
              class="edge-path"
              :d="edge.path"
              :stroke="edge.color"
              :marker-end="edge.color === '#e11d48' ? 'url(#arrow-orange)' : 'url(#arrow-blue)'"
            />
            <g class="edge-label" :transform="`translate(${edge.labelX}, ${edge.labelY})`">
              <rect x="-18" y="-10" width="36" height="20" rx="10" />
              <text text-anchor="middle" dy="4">{{ edge.label }}</text>
            </g>
          </g>
        </svg>

        <div
          v-for="node in renderedNodes"
          :key="node.id"
          class="topology-node"
          :class="{ dim: isNodeDim(node), focus: isNodeFocus(node) }"
          :style="{
            width: `${node.width}px`,
            height: `${node.height}px`,
            transform: `translate(${node.x - node.width / 2}px, ${node.y - node.height / 2}px)`
          }"
          @pointerdown.stop="onNodePointerDown($event, node)"
          @click.stop="onNodeClick(node)"
        >
          <div class="topology-node-header" :style="{ background: node.headerColor }">
            <span class="topology-node-title">{{ node.title }}</span>
          </div>
          <div class="topology-node-body">
            <p v-for="(line, index) in node.bodyLines" :key="`${node.id}-${index}`">{{ line }}</p>
          </div>
        </div>
      </div>
    </div>

    <div v-if="!hasGraphData" class="empty-tip">暂无拓扑数据，请先检查主机、应用、依赖数据是否已录入</div>
  </div>
</template>

<style scoped>
.topology-shell {
  position: relative;
  height: 760px;
  border: 1px solid #d6e2ef;
  border-radius: 18px;
  overflow: hidden;
  background:
    radial-gradient(circle at 15% 0%, rgba(74, 128, 182, 0.18), transparent 34%),
    radial-gradient(circle at 85% 100%, rgba(92, 150, 206, 0.14), transparent 30%),
    linear-gradient(180deg, #f9fbff 0%, #eef4fa 100%);
}

.topology-toolbar {
  position: absolute;
  inset: 14px 16px auto 16px;
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
  z-index: 8;
  pointer-events: none;
}

.lane-legend {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  pointer-events: auto;
}

.lane-chip {
  border: 1px solid;
  border-radius: 999px;
  height: 28px;
  padding: 0 12px;
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
  backdrop-filter: blur(8px);
  transition: transform 0.18s ease, box-shadow 0.18s ease;
}

.lane-chip:hover {
  transform: translateY(-1px);
  box-shadow: 0 6px 14px rgba(19, 53, 86, 0.14);
}

.lane-chip.active {
  box-shadow: inset 0 0 0 1px rgba(17, 24, 39, 0.35), 0 8px 18px rgba(19, 53, 86, 0.16);
}

.topology-actions {
  display: flex;
  flex-direction: column;
  gap: 10px;
  pointer-events: auto;
}

.topology-actions :deep(.el-button + .el-button) {
  margin-left: 0;
}

.topology-action-btn {
  width: 36px;
  height: 36px;
  font-size: 18px;
  border: 1px solid #b4c9df;
  background: rgba(248, 251, 255, 0.9);
  color: #21466f;
  box-shadow: 0 10px 20px rgba(26, 61, 93, 0.12);
}

.topology-viewport {
  position: absolute;
  inset: 0;
  overflow: hidden;
  cursor: grab;
}

.topology-viewport.panning {
  cursor: grabbing;
}

.topology-scene {
  position: absolute;
  inset: 0;
  transform-origin: 0 0;
  will-change: transform;
}

.edge-layer {
  position: absolute;
  inset: 0;
  overflow: visible;
  pointer-events: none;
}

.edge-path {
  fill: none;
  stroke-width: 2.4;
  opacity: 0.88;
}

.edge-group.dim {
  opacity: 0.18;
}

.edge-group.focus .edge-path {
  stroke-width: 3;
  opacity: 1;
}

.edge-label rect {
  fill: rgba(248, 251, 255, 0.96);
  stroke: rgba(148, 171, 196, 0.35);
}

.edge-label text {
  fill: #62788f;
  font-size: 11px;
  font-weight: 700;
}

.topology-node {
  position: absolute;
  border: 1px solid rgba(184, 204, 226, 0.9);
  border-radius: 18px;
  overflow: hidden;
  background: rgba(255, 255, 255, 0.96);
  box-shadow: 0 18px 34px rgba(23, 55, 86, 0.14);
  cursor: move;
  user-select: none;
  transition: box-shadow 0.18s ease, opacity 0.18s ease, transform 0.18s ease;
}

.topology-node:hover {
  box-shadow: 0 22px 40px rgba(23, 55, 86, 0.18);
}

.topology-node.dim {
  opacity: 0.22;
}

.topology-node.focus {
  box-shadow: 0 0 0 2px rgba(57, 111, 168, 0.18), 0 26px 42px rgba(23, 55, 86, 0.2);
}

.topology-node-header {
  height: 38px;
  display: flex;
  align-items: center;
  padding: 0 14px;
  color: #fff;
}

.topology-node-title {
  font-size: 13px;
  font-weight: 800;
  letter-spacing: 0.01em;
}

.topology-node-body {
  padding: 14px;
  display: grid;
  gap: 9px;
}

.topology-node-body p {
  margin: 0;
  color: #465d74;
  font-size: 12px;
  line-height: 1.45;
}

.empty-tip {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  z-index: 9;
  color: #5d728a;
  font-size: 14px;
  padding: 10px 14px;
  border-radius: 999px;
  background: rgba(245, 250, 255, 0.9);
  border: 1px solid rgba(142, 171, 201, 0.55);
}
</style>
