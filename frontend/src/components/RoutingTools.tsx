import { useState } from "react";
import { Upload, Download, Globe } from "lucide-react";
import type { AppSettings, RouteGraph, RouteNode, RoutingRule } from "../types";
import { ImportRouting, ExportRouting } from "../backend";
import { push } from "./Toasts";
import ClientSelect from "./ClientSelect";
import { confirmAction } from "./ClientConfirm";
const presets = [
  {
    id: "russia",
    label: "Bypass Russia",
    hint: "Российские сайты и IP — напрямую",
  },
  {
    id: "ads",
    label: "Блокировка рекламы",
    hint: "Рекламные домены — блокировать",
  },
  {
    id: "media",
    label: "Медиа через VPN",
    hint: "YouTube, Netflix, Discord и Telegram",
  },
  { id: "all", label: "Весь трафик через VPN" },
];
const btn =
  "no-drag flex h-8 shrink-0 items-center gap-1.5 rounded-lg border border-border bg-surface px-3 text-xs text-text-muted transition hover:bg-surface-2 hover:text-text disabled:opacity-40";
export default function RoutingTools({
  settings,
  disabled,
  onChange,
}: {
  settings: AppSettings;
  disabled: boolean;
  onChange: (s: AppSettings) => void;
}) {
  const [busy, setBusy] = useState(false);
  const io = async (importing: boolean) => {
    setBusy(true);
    try {
      if (importing) {
        const next = await ImportRouting();
        if (next) {
          onChange(next);
          push("Маршрутизация импортирована", "ok");
        }
      } else {
        const path = await ExportRouting();
        if (path)
          push("Маршрутизация экспортирована", "ok", { description: path });
      }
    } catch (e) {
      push(String(e), "error");
    } finally {
      setBusy(false);
    }
  };
  const apply = async (id: string) => {
    if (!id || disabled || busy) return;
    const preset = presets.find((p) => p.id === id);
    if (!preset) return;
    if (
      !(await confirmAction({
        title: preset.label,
        description:
          settings.routingMode === "pro"
            ? "Заменить текущий граф этим пресетом? Правила Easy сохранятся."
            : "Заменить текущие правила Easy этим пресетом? Граф Pro сохранится.",
        confirmLabel: "Применить пресет",
      }))
    )
      return;
    const nodes: RouteNode[] = [];
    const add = (
      type: RouteNode["type"],
      values: string[],
      action: RouteNode["action"],
      name: string,
    ) =>
      nodes.push({
        id: crypto.randomUUID(),
        type,
        values,
        action,
        name,
        x: 40,
        y: 40 + nodes.length * 160,
      });
    if (id === "russia") {
      add("geosite", ["category-ru", "ru"], "direct", "Российские сайты");
      add("geoip", ["ru"], "direct", "Российские IP");
    }
    if (id === "ads") add("geosite", ["category-ads-all"], "block", "Реклама");
    if (id === "media")
      add(
        "geosite",
        ["youtube", "netflix", "discord", "telegram"],
        "proxy",
        "Медиа через VPN",
      );
    const graph: RouteGraph = {
      nodes,
      final: id === "media" ? "direct" : "proxy",
      layout: {
        final: { x: 40, y: 40 + nodes.length * 160 },
        proxy: { x: 430, y: 40 },
        direct: { x: 430, y: 210 },
        block: { x: 430, y: 380 },
      },
    };
    if (settings.routingMode === "pro") {
      onChange({ ...settings, graph });
    } else {
      const rules: RoutingRule[] = nodes.flatMap(node => node.values.map(value => ({
        type: node.type as RoutingRule["type"], value, action: node.action as RoutingRule["action"],
      })));
      onChange({ ...settings, rules, simpleFinal: graph.final || "proxy" });
    }
    push(preset.label, "ok", { description: "Пресет маршрутизации применён" });
  };
  return (
    <div
      className={`mx-auto flex w-full flex-wrap items-center gap-2 ${settings.routingMode === "pro" ? "" : "max-w-2xl"}`}
    >
      <ClientSelect
        label="Пресет маршрутизации"
        value=""
        options={presets}
        placeholder="Пресеты маршрутизации"
        icon={<Globe size={13} />}
        align="left"
        className="h-8 w-56 py-0"
        disabled={disabled || busy}
        onChange={apply}
      />
      <div className="ml-auto flex items-center gap-2">
        <button
          className={btn}
          disabled={disabled || busy}
          onClick={() => io(true)}
        >
          <Upload size={13} />
          Импорт
        </button>
        <button className={btn} disabled={busy} onClick={() => io(false)}>
          <Download size={13} />
          Экспорт
        </button>
      </div>
    </div>
  );
}
