import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import {
  Globe2,
  AtSign,
  Type,
  Regex,
  Network,
  Hash,
  AppWindow,
  FolderCode,
  Cable,
  Radio,
  Globe,
  Earth,
  Ellipsis,
  Ban,
  ArrowLeftRight,
  Shield,
  Plus,
  X,
  Trash2,
  ChevronUp,
  ChevronDown,
  MoreHorizontal,
  Pencil,
  LayoutGrid,
  Crosshair,
  Minus,
  Lock,
  Copy,
  Monitor,
} from "lucide-react";
import type {
  AppSettings,
  NodeType,
  Point,
  RouteAction,
  RouteGraph,
  RouteNode,
  RoutingRule,
} from "../types";
import { ProcessPicker } from "./RoutingView";

// RouteCanvas is the Pro routing editor: matcher nodes on the left are wired
// to action nodes on the right. A node's wire is its action, its position in
// the list is its priority (the first match decides), and "Остальное" takes
// whatever nothing else matched. Everything is persisted in settings.graph and
// turned into the core's route rules on connect.

// ---------------------------------------------------------------- metadata

interface TypeMeta {
  label: string;
  hint: string;
  icon: React.ReactNode;
  color: string;
  placeholder: string;
  suggest?: string[];
  validate?: (v: string) => string | null;
}

const isIP = (v: string) => /^[0-9a-fA-F:.]+(\/\d{1,3})?$/.test(v) && /[.:]/.test(v);

export const NODE_TYPES: Record<NodeType, TypeMeta> = {
  domain: {
    label: "Домены",
    hint: "Сайты и поддомены",
    icon: <Globe2 size={15} />,
    color: "#f472b6",
    placeholder: "example.com",
    validate: (v) =>
      isIP(v) && !/[a-z]/i.test(v)
        ? "Это IP — для него нода «IP / Сети»"
        : /^(\*\.)?[a-z0-9.*_-]+\.[a-z0-9-]{2,}$/i.test(v)
          ? null
          : "Домен вида example.com",
  },
  domain_full: {
    label: "Точный домен",
    hint: "Только этот адрес",
    icon: <AtSign size={15} />,
    color: "#fb7185",
    placeholder: "api.example.com",
    validate: (v) => (/^[a-z0-9._-]+\.[a-z0-9-]{2,}$/i.test(v) ? null : "Домен вида api.example.com"),
  },
  domain_keyword: {
    label: "Ключевые слова",
    hint: "Домен содержит слово",
    icon: <Type size={15} />,
    color: "#e879f9",
    placeholder: "google",
    validate: (v) => (/\s/.test(v) ? "Без пробелов" : null),
  },
  domain_regex: {
    label: "Regex доменов",
    hint: "Регулярное выражение",
    icon: <Regex size={15} />,
    color: "#c084fc",
    placeholder: "^cdn\\d+\\.example\\.com$",
    validate: (v) => {
      try {
        new RegExp(v);
        return null;
      } catch {
        return "Некорректное выражение";
      }
    },
  },
  ip: {
    label: "IP / Сети",
    hint: "IP-адреса и CIDR",
    icon: <Network size={15} />,
    color: "#d4e157",
    placeholder: "2.56.24.0/22",
    validate: (v) => (isIP(v) ? null : "IP или CIDR, напр. 10.0.0.0/8"),
  },
  port: {
    label: "Порты",
    hint: "Порт или диапазон",
    icon: <Hash size={15} />,
    color: "#67e8f9",
    placeholder: "443 или 27000-27100",
    validate: (v) => {
      const m = v.match(/^(\d{1,5})(?:\s*[-:]\s*(\d{1,5}))?$/);
      if (!m) return "Порт или диапазон 1000-2000";
      const a = +m[1];
      const b = m[2] ? +m[2] : a;
      return a < 1 || b > 65535 || a > b ? "Порты от 1 до 65535" : null;
    },
  },
  process: {
    label: "Приложения",
    hint: "По имени процесса",
    icon: <AppWindow size={15} />,
    color: "#a5b4fc",
    placeholder: "javaw.exe",
    validate: (v) => (/\.exe$/i.test(v) ? null : "Имя процесса, напр. steam.exe"),
  },
  process_path: {
    label: "Пути приложений",
    hint: "Полный путь к .exe",
    icon: <FolderCode size={15} />,
    color: "#93c5fd",
    placeholder: "C:\\Games\\game.exe",
    validate: (v) => (/^[a-z]:\\.+\.exe$/i.test(v) ? null : "Путь вида C:\\Папка\\app.exe"),
  },
  network: {
    label: "Сеть",
    hint: "TCP или UDP",
    icon: <Cable size={15} />,
    color: "#fcd34d",
    placeholder: "udp",
    suggest: ["tcp", "udp"],
    validate: (v) => (/^(tcp|udp)$/i.test(v) ? null : "tcp или udp"),
  },
  protocol: {
    label: "Протоколы",
    hint: "Определяются сниффингом",
    icon: <Radio size={15} />,
    color: "#fdba74",
    placeholder: "bittorrent",
    suggest: ["tls", "http", "quic", "bittorrent", "stun", "dtls", "ssh", "rdp", "ntp"],
    validate: (v) =>
      /^(tls|http|quic|bittorrent|stun|dtls|ssh|rdp|ntp|dns)$/i.test(v) ? null : "Протокол из списка",
  },
  geosite: {
    label: "GeoSite",
    hint: "Готовые списки сайтов",
    icon: <Globe size={15} />,
    color: "#5eead4",
    placeholder: "category-ads-all",
    suggest: ["category-ads-all", "youtube", "google", "telegram", "discord", "openai", "netflix", "steam", "category-ru", "ru"],
    validate: (v) => (/^[a-z0-9!@._-]+$/i.test(v.replace(/^geosite:/, "")) ? null : "Имя списка, напр. youtube"),
  },
  geoip: {
    label: "GeoIP",
    hint: "Страны по IP",
    icon: <Earth size={15} />,
    color: "#6ee7b7",
    placeholder: "ru",
    suggest: ["ru", "us", "de", "nl", "cn", "telegram", "google", "netflix"],
    validate: (v) => (/^[a-z0-9!@._-]+$/i.test(v.replace(/^geoip:/, "")) ? null : "Код страны, напр. ru"),
  },
};

// What the add dialog asks for, per node type, with an example line.
const FIELD: Record<NodeType, { label: string; example: string; list: boolean }> = {
  domain: { label: "Домен или wildcard", example: "youtube.com, *.google.com", list: true },
  domain_full: { label: "Точный домен", example: "api.example.com", list: true },
  domain_keyword: { label: "Слово в домене", example: "google, ads", list: true },
  domain_regex: { label: "Регулярное выражение", example: "^cdn\\d+\\.example\\.com$", list: false },
  ip: { label: "IP-адрес или CIDR", example: "192.168.1.0/24, 10.0.0.1", list: true },
  port: { label: "Порт или диапазон", example: "443, 27000-27100", list: true },
  process: { label: "Имя процесса", example: "chrome.exe, telegram.exe", list: true },
  process_path: { label: "Путь к приложению", example: "C:\\Games\\game.exe", list: false },
  network: { label: "Сеть", example: "tcp, udp", list: true },
  protocol: { label: "Протокол", example: "bittorrent, quic", list: true },
  geosite: { label: "Список GeoSite", example: "youtube, category-ads-all", list: true },
  geoip: { label: "Страна или список GeoIP", example: "ru, us", list: true },
};

// The + menu, grouped by what a node matches on.
const TYPE_GROUPS: { title: string; types: NodeType[] }[] = [
  { title: "Сайты", types: ["domain", "domain_keyword", "domain_regex", "geosite"] },
  { title: "Адреса и порты", types: ["ip", "geoip", "port"] },
  { title: "Приложения", types: ["process", "process_path"] },
  { title: "Трафик", types: ["network", "protocol"] },
];

interface ActionMeta {
  label: string;
  hint: string;
  body: string;
  icon: React.ReactNode;
  color: string;
}

const ACTIONS: Record<RouteAction, ActionMeta> = {
  block: {
    label: "Блокировать",
    hint: "Обрывать соединение",
    body: "Соединение блокируется",
    icon: <Ban size={15} />,
    color: "var(--color-danger)",
  },
  direct: {
    label: "Напрямую",
    hint: "Обход туннеля",
    body: "Идёт мимо туннеля",
    icon: <ArrowLeftRight size={15} />,
    color: "var(--color-ok)",
  },
  proxy: {
    label: "Через VPN",
    hint: "Шифруется в туннеле",
    body: "Трафик шифруется",
    icon: <Shield size={15} />,
    color: "var(--color-accent)",
  },
};

const ACTION_ORDER: RouteAction[] = ["block", "direct", "proxy"];

// Geometry. Ports sit at the header's middle, so wires never depend on how
// tall a node's value list has grown.
const NODE_W = 244;
const ACTION_W = 208;
const PORT_Y = 27;
const FINAL_ID = "final";

const DEFAULT_LAYOUT: Record<string, Point> = {
  final: { x: 40, y: 520 },
  block: { x: 640, y: 40 },
  direct: { x: 640, y: 200 },
  proxy: { x: 640, y: 380 },
};

// ------------------------------------------------------------------ graph ops

const newId = () => Math.random().toString(36).slice(2, 10);

// Action nodes exist only once the user adds them; an action is on the canvas
// when the layout has a place for it.
const hasAction = (g: RouteGraph, a: RouteAction | "") => !!a && !!g.layout[a];
const presentActions = (g: RouteGraph) => ACTION_ORDER.filter((a) => g.layout[a]);

// actionSpot places a new action node right of everything, below the actions
// already there.
function actionSpot(g: RouteGraph): Point {
  const right = Math.max(
    40 + NODE_W,
    ...g.nodes.map((n) => n.x + NODE_W),
    (g.layout.final?.x ?? 40) + NODE_W
  );
  const own = presentActions(g).map((a) => g.layout[a]);
  const x = own.length ? Math.min(...own.map((p) => p.x)) : right + 140;
  const y = own.length ? Math.max(...own.map((p) => p.y)) + 150 : 40;
  return { x, y };
}

// graphFromRules seeds a Pro graph from the simple rule list: one node per
// (type, action) pair, so switching modes loses nothing.
export function graphFromRules(rules: RoutingRule[]): RouteGraph {
  const nodes: RouteNode[] = [];
  let y = 40;
  for (const action of ACTION_ORDER) {
    for (const type of ["ip", "domain", "process"] as NodeType[]) {
      const own = rules.filter((r) => r.type === type && r.action === action);
      if (own.length === 0) continue;
      const icons: Record<string, string> = {};
      own.forEach((r) => r.icon && (icons[r.value] = r.icon));
      nodes.push({ id: newId(), type, name: "", values: own.map((r) => r.value), icons, action, x: 40, y });
      y += 110 + own.length * 34;
    }
  }
  const layout: Record<string, Point> = { final: { x: 40, y: Math.max(y, DEFAULT_LAYOUT.final.y) } };
  for (const a of ACTION_ORDER) {
    if (a === "proxy" || nodes.some((n) => n.action === a)) layout[a] = DEFAULT_LAYOUT[a];
  }
  return { nodes, final: "proxy", layout };
}

// estimateHeight approximates a rendered node for auto-layout.
const estimateHeight = (n: RouteNode) => 104 + Math.min(n.values.length, 6) * 34;

// autoLayout stacks the matchers in priority order with the catch-all last,
// and spreads the actions down the right. Heights are measured when the nodes
// are on screen, estimated otherwise.
function autoLayout(g: RouteGraph, heights: Record<string, number> = {}): RouteGraph {
  let y = 40;
  const nodes = g.nodes.map((n) => {
    const placed = { ...n, x: 40, y };
    y += (heights[n.id] ?? estimateHeight(n)) + 24;
    return placed;
  });
  const span = Math.max(y + 120, 560);
  const layout: Record<string, Point> = { final: { x: 40, y } };
  const own = presentActions(g);
  own.forEach((a, i) => (layout[a] = { x: 420, y: span * (0.12 + (0.56 * i) / Math.max(own.length - 1, 1)) }));
  return { ...g, nodes, layout };
}

// ------------------------------------------------------------------ component

interface Props {
  settings: AppSettings;
  disabled: boolean;
  onChange: (s: AppSettings) => void;
}

type Drag =
  | { kind: "pan"; sx: number; sy: number; ox: number; oy: number }
  | { kind: "node"; id: string; sx: number; sy: number; ox: number; oy: number; moved: boolean }
  | { kind: "wire"; from: string; x: number; y: number };

export default function RouteCanvas({ settings, disabled, onChange }: Props) {
  // A local copy keeps dragging smooth; it is written back on release.
  const [graph, setGraph] = useState<RouteGraph>(() => normalize(settings.graph));
  const graphRef = useRef(graph);
  graphRef.current = graph;
  useEffect(() => setGraph(normalize(settings.graph)), [settings.graph]);

  const [view, setView] = useState({ x: 24, y: 24, z: 0.9 });
  const [drag, setDrag] = useState<Drag | null>(null);
  const [hoverWire, setHoverWire] = useState<string | null>(null);
  const cut = (id: string) => {
    setHoverWire(null);
    if (id === FINAL_ID) {
      edit((g) => ({ ...g, final: "" }));
      return;
    }
    editNode(id, (n) => ({ ...n, action: "" }));
  };
  const [adding, setAdding] = useState<string | null>(null); // node id the add dialog is for
  // Right-click menu on a node, at canvas-relative coordinates.
  const [ctx, setCtx] = useState<{ id: string; x: number; y: number; action?: boolean } | null>(null);

  useEffect(() => {
    if (!ctx) return;
    const close = () => setCtx(null);
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && close();
    window.addEventListener("mousedown", close);
    window.addEventListener("wheel", close);
    window.addEventListener("keydown", onKey);
    return () => {
      window.removeEventListener("mousedown", close);
      window.removeEventListener("wheel", close);
      window.removeEventListener("keydown", onKey);
    };
  }, [ctx]);

  // A copy lands just under the original and right after it in priority; the
  // overlap pass then makes room for it.
  const duplicate = (id: string) =>
    edit((g) => {
      const i = g.nodes.findIndex((n) => n.id === id);
      if (i < 0) return g;
      const src = g.nodes[i];
      const copy: RouteNode = {
        ...src,
        id: newId(),
        name: src.name ? `${src.name} (копия)` : "",
        values: [...src.values],
        icons: src.icons ? { ...src.icons } : undefined,
        y: src.y + 40,
      };
      const nodes = [...g.nodes];
      nodes.splice(i + 1, 0, copy);
      return { ...g, nodes };
    });

  const addAction = (a: RouteAction) => {
    setAddMenu(false);
    edit((g) => (g.layout[a] ? g : { ...g, layout: { ...g.layout, [a]: actionSpot(g) } }));
  };

  // Removing an action node unwires everything that fed it.
  const removeAction = (a: RouteAction) =>
    edit((g) => {
      const layout = { ...g.layout };
      delete layout[a];
      return {
        ...g,
        layout,
        final: g.final === a ? "" : g.final,
        nodes: g.nodes.map((n) => (n.action === a ? { ...n, action: "" } : n)),
      };
    });

  const addValues = (id: string, values: string[], icons: Record<string, string>) =>
    editNode(id, (n) => {
      const have = new Set(n.values.map((v) => v.toLowerCase()));
      const fresh = values.filter((v) => !have.has(v.toLowerCase()));
      const own = Object.fromEntries(Object.entries(icons).filter(([k]) => fresh.includes(k)));
      return {
        ...n,
        values: [...n.values, ...fresh],
        icons: Object.keys(own).length ? { ...(n.icons ?? {}), ...own } : n.icons,
      };
    });
  const [addMenu, setAddMenu] = useState(false);
  const wrap = useRef<HTMLDivElement>(null);

  const commit = useCallback(
    (g: RouteGraph) => {
      setGraph(g);
      onChange({ ...settings, graph: g });
    },
    [onChange, settings]
  );

  const edit = (fn: (g: RouteGraph) => RouteGraph) => {
    if (disabled) return;
    commit(fn(graphRef.current));
  };
  const editNode = (id: string, fn: (n: RouteNode) => RouteNode) =>
    edit((g) => ({ ...g, nodes: g.nodes.map((n) => (n.id === id ? fn(n) : n)) }));

  // Screen → canvas coordinates.
  const toWorld = (cx: number, cy: number) => {
    const r = wrap.current!.getBoundingClientRect();
    return { x: (cx - r.left - view.x) / view.z, y: (cy - r.top - view.y) / view.z };
  };

  // ---- pointer handling (window-level so a drag survives leaving a node)
  useEffect(() => {
    if (!drag) return;
    const move = (e: MouseEvent) => {
      if (drag.kind === "pan") {
        setView((v) => ({ ...v, x: drag.ox + e.clientX - drag.sx, y: drag.oy + e.clientY - drag.sy }));
      } else if (drag.kind === "node") {
        const dx = (e.clientX - drag.sx) / view.z;
        const dy = (e.clientY - drag.sy) / view.z;
        if (!drag.moved && Math.abs(dx) + Math.abs(dy) > 2) setDrag({ ...drag, moved: true });
        setGraph((g) => moveNode(g, drag.id, drag.ox + dx, drag.oy + dy));
      } else {
        const p = toWorld(e.clientX, e.clientY);
        setDrag({ ...drag, x: p.x, y: p.y });
      }
    };
    const up = (e: MouseEvent) => {
      if (drag.kind === "node" && drag.moved) commit(graphRef.current);
      if (drag.kind === "wire") {
        // Dropped on an action node (or its body): wire it there. Dropped on
        // empty canvas: the node is unwired.
        const el = document.elementFromPoint(e.clientX, e.clientY)?.closest("[data-action]");
        const action = (el?.getAttribute("data-action") ?? "") as RouteAction | "";
        const g = graphRef.current;
        if (drag.from === FINAL_ID) {
          commit({ ...g, final: action });
        } else {
          commit({ ...g, nodes: g.nodes.map((n) => (n.id === drag.from ? { ...n, action } : n)) });
        }
      }
      setDrag(null);
    };
    window.addEventListener("mousemove", move);
    window.addEventListener("mouseup", up);
    return () => {
      window.removeEventListener("mousemove", move);
      window.removeEventListener("mouseup", up);
    };
  }, [drag, view.z, commit]);

  // Wheel zooms around the cursor.
  const onWheel = (e: React.WheelEvent) => {
    const r = wrap.current!.getBoundingClientRect();
    const mx = e.clientX - r.left;
    const my = e.clientY - r.top;
    setView((v) => {
      const z = clamp(v.z * (e.deltaY < 0 ? 1.1 : 1 / 1.1), 0.4, 1.6);
      return { z, x: mx - ((mx - v.x) * z) / v.z, y: my - ((my - v.y) * z) / v.z };
    });
  };

  const startPan = (e: React.MouseEvent) => {
    if (e.button !== 0 && e.button !== 1) return;
    setAddMenu(false);
    setDrag({ kind: "pan", sx: e.clientX, sy: e.clientY, ox: view.x, oy: view.y });
  };

  const startNode = (id: string, e: React.MouseEvent) => {
    if (e.button !== 0) return;
    e.stopPropagation();
    if (disabled) return;
    const p = posOf(graphRef.current, id);
    setDrag({ kind: "node", id, sx: e.clientX, sy: e.clientY, ox: p.x, oy: p.y, moved: false });
  };

  const startWire = (from: string, e: React.MouseEvent) => {
    e.stopPropagation();
    if (disabled || e.button !== 0) return;
    const p = toWorld(e.clientX, e.clientY);
    setDrag({ kind: "wire", from, x: p.x, y: p.y });
  };

  const fit = () => {
    const r = wrap.current?.getBoundingClientRect();
    if (!r) return;
    // Read through the ref: fit also runs right after a layout change, before
    // this closure's graph has caught up.
    const g = graphRef.current;
    const pts = [
      ...g.nodes.map((n) => ({ x: n.x, y: n.y, w: NODE_W, h: estimateHeight(n) })),
      ...["final", ...presentActions(g)].map((k) => ({ ...g.layout[k], w: NODE_W, h: 120 })),
    ];
    const minX = Math.min(...pts.map((p) => p.x));
    const minY = Math.min(...pts.map((p) => p.y));
    const maxX = Math.max(...pts.map((p) => p.x + p.w));
    const maxY = Math.max(...pts.map((p) => p.y + p.h));
    // The toolbar floats over the top edge, so the frame starts below it.
    const top = 60;
    const pad = 24;
    const z = clamp(
      Math.min((r.width - pad * 2) / (maxX - minX), (r.height - top - pad) / (maxY - minY)),
      0.4,
      1.2
    );
    setView({ z, x: pad - minX * z + (r.width - pad * 2 - (maxX - minX) * z) / 2, y: top - minY * z });
  };

  useEffect(() => {
    // Frame the graph once on open.
    const t = setTimeout(fit, 0);
    return () => clearTimeout(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // A new node joins the end of the matcher column — where it also sits in
  // priority — taking the catch-all's place and pushing it down.
  const addNode = (type: NodeType) => {
    setAddMenu(false);
    if (disabled) return;
    const g = graphRef.current;
    const at = { ...g.layout.final };
    const node: RouteNode = { id: newId(), type, name: "", values: [], action: "", x: at.x, y: at.y };
    commit({
      ...g,
      nodes: [...g.nodes, node],
      layout: { ...g.layout, final: { x: at.x, y: at.y + estimateHeight(node) + 28 } },
    });
    // Bring it into view if it landed below the fold.
    const r = wrap.current?.getBoundingClientRect();
    if (r) {
      setView((v) => {
        const sy = at.y * v.z + v.y;
        return sy > r.height - 220 ? { ...v, y: v.y - (sy - (r.height - 260)) } : v;
      });
    }
  };

  // measure reads the rendered height of every node, unscaled.
  const measure = () => {
    const heights: Record<string, number> = {};
    wrap.current?.querySelectorAll<HTMLElement>("[data-node-id]").forEach((el) => {
      heights[el.dataset.nodeId!] = el.offsetHeight;
    });
    return heights;
  };

  // Nodes grow as values are added. After every change, measure the real
  // heights and push down whatever a node now overlaps in its column, so a
  // growing list never slides under its neighbour. Only ever moves nodes down,
  // so it settles in one pass.
  useLayoutEffect(() => {
    if (drag?.kind === "node" || !wrap.current) return;
    const heights = measure();
    const g = graphRef.current;
    const items = [
      ...g.nodes.map((n) => ({ id: n.id, x: n.x, y: n.y })),
      { id: FINAL_ID, ...g.layout.final },
    ].sort((a, b) => a.y - b.y);
    let moved = false;
    items.forEach((it, i) => {
      for (let j = 0; j < i; j++) {
        const above = items[j];
        if (Math.abs(above.x - it.x) >= NODE_W) continue;
        const bottom = above.y + (heights[above.id] ?? 0) + 16;
        if (it.y < bottom) {
          it.y = bottom;
          moved = true;
        }
      }
    });
    if (!moved) return;
    const pos = Object.fromEntries(items.map((it) => [it.id, it.y]));
    commit({
      ...g,
      nodes: g.nodes.map((n) => ({ ...n, y: pos[n.id] })),
      layout: { ...g.layout, final: { ...g.layout.final, y: pos[FINAL_ID] } },
    });
  }, [graph, drag, commit]);

  // ---- wires
  const wires = useMemo(() => {
    const out: { id: string; from: Point; to: Point; action: RouteAction }[] = [];
    graph.nodes.forEach((n) => {
      if (!hasAction(graph, n.action)) return;
      const a = n.action as RouteAction;
      out.push({ id: n.id, from: outPort(n), to: inPort(graph, a), action: a });
    });
    const f = graph.layout.final;
    if (hasAction(graph, graph.final)) {
      out.push({
        id: FINAL_ID,
        from: { x: f.x + NODE_W, y: f.y + PORT_Y },
        to: inPort(graph, graph.final as RouteAction),
        action: graph.final as RouteAction,
      });
    }
    return out;
  }, [graph]);

  const draftFrom =
    drag?.kind === "wire"
      ? drag.from === FINAL_ID
        ? { x: graph.layout.final.x + NODE_W, y: graph.layout.final.y + PORT_Y }
        : outPort(graph.nodes.find((n) => n.id === drag.from)!)
      : null;

  return (
    <div className="route-canvas relative min-h-0 flex-1 overflow-hidden border-t border-border bg-bg">
      {/* Dotted backdrop moves with the view so panning reads as motion. */}
      <div
        ref={wrap}
        onMouseDown={startPan}
        onWheel={onWheel}
        className={`no-drag absolute inset-0 ${drag?.kind === "pan" ? "cursor-grabbing" : "cursor-grab"}`}
        style={{
          backgroundImage:
            "radial-gradient(circle, color-mix(in srgb, var(--color-text-faint) 35%, transparent) 1px, transparent 1.2px)",
          backgroundSize: `${22 * view.z}px ${22 * view.z}px`,
          backgroundPosition: `${view.x}px ${view.y}px`,
        }}
      >
        <div
          className="absolute left-0 top-0 origin-top-left"
          style={{ transform: `translate(${view.x}px, ${view.y}px) scale(${view.z})` }}
        >
          <svg className="pointer-events-none absolute left-0 top-0 overflow-visible" width="1" height="1">
            {wires.map((w) => (
              <g key={w.id} className="pointer-events-auto">
                {/* Wide transparent twin makes the thin wire easy to hit. */}
                {/* Over a wire the cursor turns into scissors and a click
                    cuts it. */}
                <path
                  d={curve(w.from, w.to)}
                  stroke="transparent"
                  strokeWidth={16}
                  fill="none"
                  className={disabled ? "" : "cursor-scissors"}
                  onMouseEnter={() => setHoverWire(w.id)}
                  onMouseLeave={() => setHoverWire(null)}
                  onMouseDown={(e) => e.stopPropagation()}
                  onClick={() => !disabled && cut(w.id)}
                />
                <path
                  d={curve(w.from, w.to)}
                  stroke={
                    hoverWire === w.id && !disabled
                      ? "var(--color-danger)"
                      : ACTIONS[w.action].color
                  }
                  strokeOpacity={hoverWire === w.id ? 1 : 0.7}
                  strokeWidth={hoverWire === w.id ? 2.4 : 1.6}
                  strokeDasharray="5 5"
                  fill="none"
                  pointerEvents="none"
                  className="route-wire"
                />
                <circle cx={w.to.x} cy={w.to.y} r={3} fill={ACTIONS[w.action].color} pointerEvents="none" />
              </g>
            ))}
            {drag?.kind === "wire" && draftFrom && (
              <path
                d={curve(draftFrom, { x: drag.x, y: drag.y })}
                stroke="var(--color-text-muted)"
                strokeWidth={1.6}
                strokeDasharray="5 5"
                fill="none"
              />
            )}
          </svg>

          {graph.nodes.map((n, i) => (
            <MatcherNode
              key={n.id}
              node={n}
              index={i}
              count={graph.nodes.length}
              disabled={disabled}
              wiring={drag?.kind === "wire" && drag.from === n.id}
              onGrab={(e) => startNode(n.id, e)}
              onWire={(e) => startWire(n.id, e)}
              onChange={(fn) => editNode(n.id, fn)}
              onAdd={() => setAdding(n.id)}
              onContext={(e) => {
                if (disabled) return;
                const r = e.currentTarget.closest(".route-canvas")!.getBoundingClientRect();
                setCtx({ id: n.id, x: e.clientX - r.left, y: e.clientY - r.top });
              }}
              onMove={(dir) =>
                edit((g) => {
                  const nodes = [...g.nodes];
                  const j = i + dir;
                  if (j < 0 || j >= nodes.length) return g;
                  [nodes[i], nodes[j]] = [nodes[j], nodes[i]];
                  return { ...g, nodes };
                })
              }
              onDelete={() => edit((g) => ({ ...g, nodes: g.nodes.filter((x) => x.id !== n.id) }))}
            />
          ))}

          <FinalNode
            pos={graph.layout.final}
            wired={hasAction(graph, graph.final)}
            disabled={disabled}
            wiring={drag?.kind === "wire" && drag.from === FINAL_ID}
            onGrab={(e) => startNode(FINAL_ID, e)}
            onWire={(e) => startWire(FINAL_ID, e)}
          />

          {presentActions(graph).map((a) => (
            <ActionNode
              key={a}
              action={a}
              pos={graph.layout[a]}
              target={drag?.kind === "wire"}
              onGrab={(e) => startNode(a, e)}
              onContext={(e) => {
                if (disabled) return;
                const r = e.currentTarget.closest(".route-canvas")!.getBoundingClientRect();
                setCtx({ id: a, x: e.clientX - r.left, y: e.clientY - r.top, action: true });
              }}
            />
          ))}
        </div>
      </div>

      {/* Toolbar */}
      <div className="pointer-events-none absolute inset-x-3 top-3 flex items-start justify-between gap-2">
        <div className="pointer-events-auto relative flex items-center gap-1.5">
          <ToolbarButton
            onClick={() => {
              edit((g) => autoLayout(g, measure()));
              setTimeout(fit, 0);
            }}
            disabled={disabled}
            title="Разложить по порядку"
          >
            <LayoutGrid size={14} />
          </ToolbarButton>
          <ToolbarButton onClick={fit} title="Показать всё">
            <Crosshair size={14} />
          </ToolbarButton>
          <ToolbarButton onClick={() => setView((v) => ({ ...v, z: clamp(v.z / 1.15, 0.4, 1.6) }))} title="Отдалить">
            <Minus size={14} />
          </ToolbarButton>
          <span className="w-10 text-center font-mono text-[11px] text-text-faint">{Math.round(view.z * 100)}%</span>
          <ToolbarButton onClick={() => setView((v) => ({ ...v, z: clamp(v.z * 1.15, 0.4, 1.6) }))} title="Приблизить">
            <Plus size={14} />
          </ToolbarButton>

        </div>

        {disabled && (
          <div className="pointer-events-auto flex items-center gap-1.5 rounded-lg border border-border bg-surface/90 px-3 py-1.5 text-[11px] text-text-faint backdrop-blur">
            <Lock size={12} /> Редактирование заблокировано, пока туннель активен
          </div>
        )}
      </div>

      {/* Everything that can be added lives behind the + in the corner. */}
      <div className="absolute bottom-4 right-4 z-30 flex flex-col items-end gap-2">
        {addMenu && (
          <div
            className="animate-pop w-[440px] origin-bottom-right overflow-hidden rounded-[var(--radius-lg)] border border-border bg-surface-2/95 shadow-2xl backdrop-blur-md"
            onMouseDown={(e) => e.stopPropagation()}
          >
            <div className="border-b border-border px-4 py-2.5">
              <div className="text-sm font-medium text-text">Добавить</div>
              <div className="text-[11px] text-text-faint">Правила встают в конец списка, с самым низким приоритетом</div>
            </div>
            <div className="flex max-h-[min(420px,calc(100vh-280px))] flex-col gap-3 overflow-y-auto p-2">
              <div>
                <div className="px-2 pb-1 text-[10px] font-medium uppercase tracking-wider text-text-faint">
                  Действия
                </div>
                <div className="grid grid-cols-2 gap-1">
                  {(["proxy", "direct", "block"] as RouteAction[]).map((a) => {
                    const there = hasAction(graph, a);
                    return (
                      <button
                        key={a}
                        disabled={there}
                        onClick={() => addAction(a)}
                        className="no-drag flex items-center gap-2.5 rounded-lg px-2.5 py-2 text-left transition hover:bg-surface disabled:cursor-default disabled:opacity-40 disabled:hover:bg-transparent"
                      >
                        <span
                          className="grid h-7 w-7 shrink-0 place-items-center rounded-md"
                          style={{
                            color: ACTIONS[a].color,
                            background: `color-mix(in srgb, ${ACTIONS[a].color} 14%, transparent)`,
                          }}
                        >
                          {ACTIONS[a].icon}
                        </span>
                        <span className="min-w-0">
                          <span className="block text-sm text-text">{ACTIONS[a].label}</span>
                          <span className="block truncate text-[11px] text-text-faint">
                            {there ? "Уже на холсте" : ACTIONS[a].hint}
                          </span>
                        </span>
                      </button>
                    );
                  })}
                </div>
              </div>
              {TYPE_GROUPS.map((g) => (
                <div key={g.title}>
                  <div className="px-2 pb-1 text-[10px] font-medium uppercase tracking-wider text-text-faint">
                    {g.title}
                  </div>
                  <div className="grid grid-cols-2 gap-1">
                    {g.types.map((t) => (
                      <button
                        key={t}
                        onClick={() => addNode(t)}
                        className="no-drag flex items-center gap-2.5 rounded-lg px-2.5 py-2 text-left transition hover:bg-surface"
                      >
                        <span
                          className="grid h-7 w-7 shrink-0 place-items-center rounded-md"
                          style={{
                            color: NODE_TYPES[t].color,
                            background: `color-mix(in srgb, ${NODE_TYPES[t].color} 14%, transparent)`,
                          }}
                        >
                          {NODE_TYPES[t].icon}
                        </span>
                        <span className="min-w-0">
                          <span className="block text-sm text-text">{NODE_TYPES[t].label}</span>
                          <span className="block truncate text-[11px] text-text-faint">{NODE_TYPES[t].hint}</span>
                        </span>
                      </button>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
        <button
          onClick={() => setAddMenu((o) => !o)}
          onMouseDown={(e) => e.stopPropagation()}
          disabled={disabled}
          title="Добавить ноду"
          className="no-drag grid h-12 w-12 place-items-center rounded-full bg-accent text-bg shadow-[0_8px_24px_rgb(0_0_0/0.35)] transition hover:bg-accent-soft active:scale-95 disabled:cursor-not-allowed disabled:opacity-50"
        >
          <Plus size={22} className={`transition-transform duration-200 ${addMenu ? "rotate-45" : ""}`} />
        </button>
      </div>

      {ctx && (
        <div
          className="animate-pop absolute z-40 w-44 overflow-hidden rounded-lg border border-border bg-surface-2 py-1 shadow-2xl"
          style={{ left: ctx.x, top: ctx.y }}
          onMouseDown={(e) => e.stopPropagation()}
        >
          {!ctx.action && (
            <>
              <MenuItem icon={<Copy size={13} />} onClick={() => (duplicate(ctx.id), setCtx(null))}>
                Дублировать
              </MenuItem>
              <div className="my-1 h-px bg-border" />
            </>
          )}
          <MenuItem
            icon={<Trash2 size={13} />}
            danger
            onClick={() => {
              if (ctx.action) removeAction(ctx.id as RouteAction);
              else edit((g) => ({ ...g, nodes: g.nodes.filter((x) => x.id !== ctx.id) }));
              setCtx(null);
            }}
          >
            Удалить
          </MenuItem>
        </div>
      )}

      {adding && graph.nodes.some((n) => n.id === adding) && (
        <AddValueDialog
          node={graph.nodes.find((n) => n.id === adding)!}
          onClose={() => setAdding(null)}
          onAdd={(values, icons) => {
            addValues(adding, values, icons);
            setAdding(null);
          }}
        />
      )}
    </div>
  );
}

// AddValueDialog asks for one or more values for a node. List-like kinds take
// several at once, separated by commas or new lines.
function AddValueDialog({
  node,
  onClose,
  onAdd,
}: {
  node: RouteNode;
  onClose: () => void;
  onAdd: (values: string[], icons: Record<string, string>) => void;
}) {
  const meta = NODE_TYPES[node.type] ?? NODE_TYPES.domain;
  const field = FIELD[node.type] ?? FIELD.domain;
  const [draft, setDraft] = useState("");
  const [err, setErr] = useState<string | null>(null);
  const [picking, setPicking] = useState(false);
  const [icons, setIcons] = useState<Record<string, string>>({});

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && !picking && onClose();
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose, picking]);

  const parts = (text: string) =>
    (field.list ? text.split(/[,\n]+/) : [text]).map((v) => v.trim()).filter(Boolean);

  const submit = () => {
    const values = parts(draft);
    if (values.length === 0) return;
    for (const v of values) {
      const problem = meta.validate?.(v) ?? null;
      if (problem) {
        setErr(values.length > 1 ? `${v}: ${problem}` : problem);
        return;
      }
    }
    onAdd(values, icons);
  };

  const suggestions = (meta.suggest ?? []).filter((x) => !node.values.includes(x) && !parts(draft).includes(x));

  return (
    <div className="fixed inset-0 z-50 grid place-items-center bg-black/55 p-6 backdrop-blur-sm" onMouseDown={onClose}>
      <div
        className="animate-view w-full max-w-[460px] rounded-2xl border border-border bg-surface p-5 shadow-2xl"
        onMouseDown={(e) => e.stopPropagation()}
      >
        <div className="mb-5 flex items-center gap-2.5">
          <span style={{ color: meta.color }}>{meta.icon}</span>
          <h2 className="flex-1 text-base font-medium text-text">{node.name || meta.label}</h2>
          <button
            onClick={onClose}
            className="no-drag grid h-8 w-8 place-items-center rounded-full bg-surface-2 text-text-muted transition hover:text-text"
          >
            <X size={15} />
          </button>
        </div>

        <label className="mb-2 block text-sm text-text-muted">{field.label}</label>
        <input
          autoFocus
          value={draft}
          placeholder={meta.placeholder}
          spellCheck={false}
          onChange={(e) => (setDraft(e.target.value), setErr(null))}
          onKeyDown={(e) => e.key === "Enter" && submit()}
          className={`w-full rounded-xl border bg-bg px-4 py-2.5 font-mono text-sm text-text outline-none transition placeholder:text-text-faint focus:border-accent/70 focus:ring-2 focus:ring-accent/20 ${
            err ? "border-danger/60" : "border-border"
          }`}
        />

        {node.type === "process" && (
          <button
            onClick={() => setPicking(true)}
            className="no-drag mt-2 flex w-full items-center justify-center gap-2 rounded-xl border border-border py-2.5 text-sm text-text transition hover:bg-surface-2"
          >
            <Monitor size={15} /> Выбрать из запущенных
          </button>
        )}

        {err ? (
          <p className="mt-2 text-xs text-danger">{err}</p>
        ) : (
          <p className="mt-2 text-xs text-text-faint">Например: {field.example}</p>
        )}

        {suggestions.length > 0 && (
          <div className="mt-3 flex flex-wrap gap-1.5">
            {suggestions.map((x) => (
              <button
                key={x}
                onClick={() => (setDraft((d) => (d.trim() ? `${d.trim().replace(/,$/, "")}, ${x}` : x)), setErr(null))}
                className="no-drag rounded-md border border-border px-2 py-0.5 font-mono text-[11px] text-text-muted transition hover:border-accent/50 hover:text-text"
              >
                {x}
              </button>
            ))}
          </div>
        )}

        <div className="mt-6 flex justify-end gap-2">
          <button onClick={onClose} className="no-drag rounded-lg px-4 py-2 text-sm font-medium text-text transition hover:bg-surface-2">
            Отмена
          </button>
          <button
            onClick={submit}
            disabled={!draft.trim()}
            className="no-drag rounded-lg bg-accent px-4 py-2 text-sm font-medium text-bg transition hover:bg-accent-soft disabled:opacity-50"
          >
            Добавить
          </button>
        </div>
      </div>

      {picking && (
        <div onMouseDown={(e) => e.stopPropagation()}>
          <ProcessPicker
            onClose={() => setPicking(false)}
            onPick={(name, icon) => {
              setDraft((d) => {
                const have = parts(d);
                return have.some((v) => v.toLowerCase() === name.toLowerCase()) ? d : [...have, name].join(", ");
              });
              if (icon) setIcons((m) => ({ ...m, [name]: icon }));
              setErr(null);
              setPicking(false);
            }}
          />
        </div>
      )}
    </div>
  );
}

// ------------------------------------------------------------------- nodes

function MatcherNode({
  node: n,
  index,
  count,
  disabled,
  wiring,
  onGrab,
  onWire,
  onChange,
  onAdd,
  onContext,
  onMove,
  onDelete,
}: {
  node: RouteNode;
  index: number;
  count: number;
  disabled: boolean;
  wiring: boolean;
  onGrab: (e: React.MouseEvent) => void;
  onWire: (e: React.MouseEvent) => void;
  onChange: (fn: (n: RouteNode) => RouteNode) => void;
  onAdd: () => void;
  onContext: (e: React.MouseEvent) => void;
  onMove: (dir: -1 | 1) => void;
  onDelete: () => void;
}) {
  const meta = NODE_TYPES[n.type] ?? NODE_TYPES.domain;
  const [menu, setMenu] = useState(false);
  const [renaming, setRenaming] = useState(false);

  return (
    <div
      className={`absolute flex flex-col rounded-[var(--radius-lg)] border bg-surface shadow-[0_6px_24px_rgb(0_0_0/0.25)] transition-[border-color] hover:z-10 focus-within:z-20 ${
        wiring ? "border-accent/60" : "border-border"
      }`}
      data-node-id={n.id}
      style={{ left: n.x, top: n.y, width: NODE_W }}
      onMouseDown={(e) => e.stopPropagation()}
      onContextMenu={(e) => {
        e.preventDefault();
        onContext(e);
      }}
    >
      <div
        className="h-[3px] rounded-t-[var(--radius-lg)]"
        style={{ background: meta.color, opacity: n.action ? 0.9 : 0.4 }}
      />
      {/* Header doubles as the drag handle. */}
      <div
        onMouseDown={onGrab}
        className={`flex items-center gap-2.5 px-3.5 pb-2 pt-2.5 ${disabled ? "" : "cursor-move"}`}
      >
        <span
          className="grid h-7 w-7 shrink-0 place-items-center rounded-md"
          style={{ color: meta.color, background: `color-mix(in srgb, ${meta.color} 14%, transparent)` }}
        >
          {meta.icon}
        </span>
        <div className="min-w-0 flex-1">
          {renaming ? (
            <input
              autoFocus
              defaultValue={n.name || meta.label}
              onMouseDown={(e) => e.stopPropagation()}
              onBlur={(e) => {
                const name = e.target.value.trim();
                onChange((x) => ({ ...x, name: name === meta.label ? "" : name }));
                setRenaming(false);
              }}
              onKeyDown={(e) => {
                if (e.key === "Enter") (e.target as HTMLInputElement).blur();
                if (e.key === "Escape") setRenaming(false);
              }}
              className="w-full rounded border border-accent/60 bg-bg px-1.5 py-0.5 text-sm text-text outline-none"
            />
          ) : (
            <div className="truncate text-sm font-medium text-text">{n.name || meta.label}</div>
          )}
          <div className="truncate text-[11px] text-text-faint">
            {n.action ? meta.hint : "Не подключено — правило выключено"}
          </div>
        </div>
        <span
          title="Приоритет: правила проверяются сверху вниз"
          className="shrink-0 rounded bg-surface-2 px-1.5 py-0.5 font-mono text-[10px] text-text-faint"
        >
          #{index + 1}
        </span>
        {!disabled && (
          <div className="relative shrink-0" onMouseDown={(e) => e.stopPropagation()}>
            <button
              onClick={() => setMenu((m) => !m)}
              className="rounded p-1 text-text-faint transition hover:bg-surface-2 hover:text-text"
            >
              <MoreHorizontal size={14} />
            </button>
            {menu && (
              <div
                className="animate-pop absolute right-0 top-7 z-30 w-44 overflow-hidden rounded-lg border border-border bg-surface-2 py-1 shadow-xl"
                onMouseLeave={() => setMenu(false)}
              >
                <MenuItem icon={<Pencil size={13} />} onClick={() => (setRenaming(true), setMenu(false))}>
                  Переименовать
                </MenuItem>
                <MenuItem icon={<ChevronUp size={13} />} disabled={index === 0} onClick={() => (onMove(-1), setMenu(false))}>
                  Выше по приоритету
                </MenuItem>
                <MenuItem icon={<ChevronDown size={13} />} disabled={index === count - 1} onClick={() => (onMove(1), setMenu(false))}>
                  Ниже по приоритету
                </MenuItem>
                <MenuItem icon={<Trash2 size={13} />} danger onClick={() => (onDelete(), setMenu(false))}>
                  Удалить ноду
                </MenuItem>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Output port */}
      <button
        onMouseDown={onWire}
        title="Тяните к действию"
        className={`absolute right-[-7px] top-[20px] h-3.5 w-3.5 rounded-full border-2 border-bg transition hover:scale-125 ${
          disabled ? "cursor-default" : "cursor-crosshair"
        }`}
        style={{ background: n.action ? ACTIONS[n.action].color : "var(--color-text-faint)" }}
      />

      <div className="flex flex-col gap-1 px-3 pb-3" onMouseDown={(e) => e.stopPropagation()}>
        <div className="flex max-h-[214px] flex-col gap-1 overflow-y-auto" onWheel={(e) => e.stopPropagation()}>
          {n.values.map((v) => (
            <div
              key={v}
              className="group/val flex items-center gap-2 rounded-md border border-border-soft bg-bg/60 px-2.5 py-1.5"
            >
              {n.icons?.[v] && <img src={n.icons[v]} alt="" className="h-4 w-4 shrink-0 object-contain" draggable={false} />}
              <span className="min-w-0 flex-1 truncate font-mono text-xs text-text">{v}</span>
              {!disabled && (
                <button
                  onClick={() => onChange((x) => ({ ...x, values: x.values.filter((y) => y !== v) }))}
                  className="shrink-0 rounded p-0.5 text-text-faint opacity-0 transition hover:text-danger group-hover/val:opacity-100"
                >
                  <X size={12} />
                </button>
              )}
            </div>
          ))}
        </div>

        {!disabled && (
          <button
            onClick={onAdd}
            className="flex items-center justify-center gap-1.5 rounded-md py-1.5 text-xs text-text-muted transition hover:bg-surface-2 hover:text-text"
          >
            <Plus size={13} /> Добавить
          </button>
        )}
      </div>
    </div>
  );
}

function FinalNode({
  pos,
  wired,
  disabled,
  wiring,
  onGrab,
  onWire,
}: {
  pos: Point;
  wired: boolean;
  disabled: boolean;
  wiring: boolean;
  onGrab: (e: React.MouseEvent) => void;
  onWire: (e: React.MouseEvent) => void;
}) {
  const color = "#fb923c";
  return (
    <div
      className={`absolute rounded-[var(--radius-lg)] border bg-surface shadow-[0_6px_24px_rgb(0_0_0/0.25)] ${
        wiring ? "border-accent/60" : "border-border"
      }`}
      data-node-id={FINAL_ID}
      style={{ left: pos.x, top: pos.y, width: NODE_W }}
      onMouseDown={(e) => e.stopPropagation()}
    >
      <div className="h-[3px] rounded-t-[var(--radius-lg)]" style={{ background: color }} />
      <div onMouseDown={onGrab} className={`flex items-center gap-2.5 px-3.5 pb-2 pt-2.5 ${disabled ? "" : "cursor-move"}`}>
        <span
          className="grid h-7 w-7 shrink-0 place-items-center rounded-md"
          style={{ color, background: `color-mix(in srgb, ${color} 14%, transparent)` }}
        >
          <Ellipsis size={15} />
        </span>
        <div className="min-w-0">
          <div className="text-sm font-medium text-text">Остальное</div>
          <div className="text-[11px] text-text-faint">
            {wired ? "Всё, что не попало в правила" : "Не подключено — пойдёт через VPN"}
          </div>
        </div>
      </div>
      <button
        onMouseDown={onWire}
        className={`absolute right-[-7px] top-[20px] h-3.5 w-3.5 rounded-full border-2 border-bg transition hover:scale-125 ${
          disabled ? "cursor-default" : "cursor-crosshair"
        }`}
        style={{ background: color }}
      />
      <div className="px-3 pb-3">
        <div className="flex items-center gap-2 rounded-md border border-border-soft bg-bg/60 px-2.5 py-1.5 text-xs text-text-muted">
          <span className="h-1.5 w-1.5 rounded-full" style={{ background: color }} />
          Трафик без совпадений
        </div>
      </div>
    </div>
  );
}

function ActionNode({
  action,
  pos,
  target,
  onGrab,
  onContext,
}: {
  action: RouteAction;
  pos: Point;
  target: boolean;
  onGrab: (e: React.MouseEvent) => void;
  onContext: (e: React.MouseEvent) => void;
}) {
  const m = ACTIONS[action];
  return (
    <div
      data-action={action}
      className={`absolute rounded-[var(--radius-lg)] border bg-surface shadow-[0_6px_24px_rgb(0_0_0/0.25)] transition ${
        target ? "border-dashed border-text-faint hover:border-solid hover:border-[var(--c)]" : "border-border"
      }`}
      style={{ left: pos.x, top: pos.y, width: ACTION_W, ["--c" as any]: m.color }}
      onMouseDown={(e) => e.stopPropagation()}
      onContextMenu={(e) => {
        e.preventDefault();
        onContext(e);
      }}
    >
      <div className="h-[3px] rounded-t-[var(--radius-lg)]" style={{ background: m.color }} />
      <span
        className="absolute left-[-6px] top-[21px] h-3 w-3 rounded-full border-2 border-bg"
        style={{ background: m.color }}
      />
      <div onMouseDown={onGrab} className="flex cursor-move items-center gap-2.5 px-3.5 pb-2 pt-2.5">
        <span
          className="grid h-7 w-7 shrink-0 place-items-center rounded-md"
          style={{ color: m.color, background: `color-mix(in srgb, ${m.color} 14%, transparent)` }}
        >
          {m.icon}
        </span>
        <div className="min-w-0">
          <div className="text-sm font-medium text-text">{m.label}</div>
          <div className="text-[11px] text-text-faint">{m.hint}</div>
        </div>
      </div>
      <div className="px-3 pb-3">
        <div className="flex items-center gap-2 rounded-md border border-border-soft bg-bg/60 px-2.5 py-1.5 text-xs text-text-muted">
          <span className="h-1.5 w-1.5 rounded-full" style={{ background: m.color }} />
          {m.body}
        </div>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------- helpers

function ToolbarButton({
  children,
  onClick,
  disabled,
  primary,
  title,
}: {
  children: React.ReactNode;
  onClick: () => void;
  disabled?: boolean;
  primary?: boolean;
  title?: string;
}) {
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      title={title}
      className={`no-drag flex h-8 items-center gap-1.5 rounded-lg border px-2.5 text-xs backdrop-blur transition disabled:cursor-not-allowed disabled:opacity-50 ${
        primary
          ? "border-accent/40 bg-accent/15 text-accent hover:bg-accent/25"
          : "border-border bg-surface/90 text-text-muted hover:bg-surface-2 hover:text-text"
      }`}
    >
      {children}
    </button>
  );
}

function MenuItem({
  icon,
  children,
  onClick,
  disabled,
  danger,
}: {
  icon: React.ReactNode;
  children: React.ReactNode;
  onClick: () => void;
  disabled?: boolean;
  danger?: boolean;
}) {
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      className={`flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs transition hover:bg-surface disabled:opacity-40 ${
        danger ? "text-danger" : "text-text-muted hover:text-text"
      }`}
    >
      {icon}
      {children}
    </button>
  );
}

const clamp = (v: number, a: number, b: number) => Math.min(b, Math.max(a, v));

// normalize fills in a graph saved before it had every field.
function normalize(g?: RouteGraph): RouteGraph {
  return {
    nodes: (g?.nodes ?? []).map((n) => ({ ...n, values: n.values ?? [] })),
    final: g?.final ?? "proxy",
    layout: { final: DEFAULT_LAYOUT.final, ...(g?.layout ?? {}) },
  };
}

function posOf(g: RouteGraph, id: string): Point {
  const n = g.nodes.find((x) => x.id === id);
  return n ? { x: n.x, y: n.y } : g.layout[id] ?? { x: 0, y: 0 };
}

function moveNode(g: RouteGraph, id: string, x: number, y: number): RouteGraph {
  if (g.nodes.some((n) => n.id === id)) {
    return { ...g, nodes: g.nodes.map((n) => (n.id === id ? { ...n, x, y } : n)) };
  }
  return { ...g, layout: { ...g.layout, [id]: { x, y } } };
}

const outPort = (n: RouteNode): Point => ({ x: n.x + NODE_W, y: n.y + PORT_Y });

function inPort(g: RouteGraph, a: RouteAction): Point {
  const p = g.layout[a] ?? DEFAULT_LAYOUT[a];
  return { x: p.x, y: p.y + PORT_Y };
}

// curve draws a horizontal S-bezier; a backwards wire loops out and around.
function curve(a: Point, b: Point): string {
  const dx = Math.max(60, Math.abs(b.x - a.x) * 0.5);
  return `M ${a.x} ${a.y} C ${a.x + dx} ${a.y}, ${b.x - dx} ${b.y}, ${b.x} ${b.y}`;
}
