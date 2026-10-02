import SegmentedControl from "./SegmentedControl";
import { useEffect, useState } from "react";
import { RefreshCw, X, Download, Cpu, Search, Pause, Play } from "lucide-react";
import {
  GetConnections,
  CloseConnection,
  GetRuntimeStats,
  GetGoroutineDump,
  CollectGarbage,
  ExportHeapProfile,
} from "../backend";
import type { Connections, RuntimeStats } from "../backend";
import { formatBytes, formatUptime } from "../format";
import { push } from "./Toasts";
const button =
  "no-drag flex items-center gap-1.5 rounded-lg border border-border bg-surface px-3 py-1.5 text-xs text-text-muted hover:text-text hover:bg-surface-2 disabled:opacity-40";
export function ConnectionsPanel({ hide = false }: { hide?: boolean }) {
  const [data, setData] = useState<Connections | null>(null),
    [error, setError] = useState(""),
    [tab, setTab] = useState("connections"),
    [q, setQ] = useState(""),
    [paused, setPaused] = useState(false);
  useEffect(() => {
    if (paused || hide) return;
    let alive = true;
    let timer: number;
    const run = async () => {
      try {
        const d = await GetConnections();
        if (alive) {
          setData(d);
          setError("");
        }
      } catch (e) {
        if (alive) setError(String(e));
      }
      if (alive) timer = window.setTimeout(run, 1000);
    };
    run();
    return () => {
      alive = false;
      clearTimeout(timer);
    };
  }, [paused, hide]);
  const close = async (id: string) => {
    try {
      await CloseConnection(id);
      setData(await GetConnections());
    } catch (e) {
      push(String(e), "error");
    }
  };
  if (hide)
    return (
      <div className="flex flex-1 items-center justify-center p-5 text-xs text-text-faint">
        Соединения скрыты в демо-режиме
      </div>
    );
  const needle = q.toLowerCase();
  const connections = (data?.connections ?? []).filter((c) =>
    JSON.stringify(c.metadata).toLowerCase().includes(needle),
  );
  const apps = (data?.applications ?? []).filter((a) =>
    a.process.toLowerCase().includes(needle),
  );
  return (
    <section className="flex min-h-0 flex-1 flex-col">
      <div className="flex flex-wrap items-center gap-2 border-b border-border px-3 py-2">
        <SegmentedControl
          label="Тип статистики"
          value={tab}
          onChange={setTab}
          options={[
            { id: "connections", label: "Активные" },
            { id: "apps", label: "Приложения" },
          ]}
        />
        <div className="ml-auto flex min-w-0 flex-1 items-center justify-end gap-2">
          <div className="flex min-w-0 max-w-56 flex-1 items-center gap-2 rounded-lg border border-border bg-bg px-2.5 py-1.5">
            <Search size={13} className="shrink-0 text-text-faint" />
            <input
              aria-label="Фильтр соединений"
              value={q}
              onChange={(e) => setQ(e.target.value)}
              placeholder="Приложение или адрес…"
              className="min-w-0 flex-1 bg-transparent text-xs text-text outline-none"
            />
          </div>
          <button
            className="grid h-8 w-8 shrink-0 place-items-center rounded-md text-text-muted hover:bg-surface-2 hover:text-text"
            aria-label={
              paused ? "Продолжить статистику" : "Приостановить статистику"
            }
            title={paused ? "Продолжить" : "Пауза"}
            onClick={() => setPaused(!paused)}
          >
            {paused ? <Play size={14} /> : <Pause size={14} />}
          </button>
        </div>
      </div>
      {error && (
        <p className="border-b border-border px-3 py-2 text-xs text-danger">
          {error}
        </p>
      )}
      <div className="min-h-0 flex-1 overflow-auto bg-bg text-xs">
        <table className="w-full table-fixed text-left">
          <colgroup>
            <col className="w-[42%]" />
            <col className="w-[18%]" />
            <col className="w-[18%]" />
            <col className="w-[22%]" />
          </colgroup>
          <thead className="sticky top-0 z-10 border-b border-border bg-surface text-[11px] text-text-faint">
            <tr>
              <th className="px-3 py-2.5 font-normal">
                {tab === "apps" ? "Приложение" : "Назначение"}
              </th>
              <th className="px-2 py-2.5 font-normal">Скачано</th>
              <th className="px-2 py-2.5 font-normal">Отдано</th>
              <th className="px-2 py-2.5 font-normal">
                {tab === "apps" ? "Активных" : "Маршрут"}
              </th>
            </tr>
          </thead>
          <tbody>
            {tab === "apps"
              ? apps.map((a) => (
                  <tr
                    key={a.process}
                    className="border-b border-border-soft hover:bg-surface/60"
                  >
                    <td className="truncate px-3 py-2.5" title={a.process}>
                      {a.process}
                    </td>
                    <td className="px-2 py-2.5 font-mono text-[11px] text-ok">
                      {formatBytes(a.download)}
                    </td>
                    <td className="px-2 py-2.5 font-mono text-[11px] text-up">
                      {formatBytes(a.upload)}
                    </td>
                    <td className="px-2 py-2.5 font-mono text-text-muted">
                      {a.connections}
                    </td>
                  </tr>
                ))
              : connections.map((c) => (
                  <tr
                    key={c.id}
                    className="border-b border-border-soft hover:bg-surface/60"
                  >
                    <td className="px-3 py-2.5">
                      <div
                        className="truncate"
                        title={`${c.metadata.host || c.metadata.destinationIP}:${c.metadata.destinationPort}`}
                      >
                        {c.metadata.host || c.metadata.destinationIP}:
                        {c.metadata.destinationPort}
                      </div>
                      <div
                        className="mt-0.5 truncate text-[10px] text-text-faint"
                        title={c.metadata.processPath}
                      >
                        {c.metadata.process ||
                          c.metadata.processPath ||
                          "Не определено"}{" "}
                        · {c.metadata.network}
                      </div>
                    </td>
                    <td className="px-2 py-2.5 font-mono text-[11px] text-ok">
                      {formatBytes(c.download)}
                    </td>
                    <td className="px-2 py-2.5 font-mono text-[11px] text-up">
                      {formatBytes(c.upload)}
                    </td>
                    <td className="px-2 py-2.5">
                      <div className="flex items-center gap-1.5">
                        <span
                          className="min-w-0 flex-1 truncate text-[11px] text-text-muted"
                          title={c.rule}
                        >
                          {c.chains?.join(" → ") || c.rule}
                        </span>
                        <button
                          aria-label="Закрыть соединение"
                          title="Закрыть соединение"
                          onClick={() => close(c.id)}
                          className="shrink-0 text-text-faint hover:text-danger"
                        >
                          <X size={12} />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
          </tbody>
        </table>
        {(tab === "apps" ? apps.length : connections.length) === 0 && (
          <div className="px-5 py-10 text-center text-xs text-text-faint">
            {q
              ? "Нет результатов по фильтру"
              : "Нет активных соединений. Подключитесь к профилю."}
          </div>
        )}
      </div>
      <div className="flex flex-wrap items-center justify-between gap-2 border-t border-border px-3 py-2 font-mono text-[10px] text-text-faint">
        <span>
          {data?.connections.length ?? 0} соединений · ↓{" "}
          {formatBytes(data?.downloadTotal ?? 0)} · ↑{" "}
          {formatBytes(data?.uploadTotal ?? 0)}
        </span>
        <span title="Трафик приложений — накопленные наблюдения за сеанс. Короткие соединения между опросами могут не попасть в статистику.">
          {paused ? "Пауза" : "Опрос · 1 с"}
        </span>
      </div>
    </section>
  );
}
export function DeveloperMonitor({ hide = false }: { hide?: boolean }) {
  const [stats, setStats] = useState<RuntimeStats | null>(null),
    [dump, setDump] = useState(""),
    [error, setError] = useState("");
  useEffect(() => {
    let alive = true;
    let timer: number;
    const run = async () => {
      try {
        const s = await GetRuntimeStats();
        if (alive) {
          setStats(s);
          setError("");
        }
      } catch (e) {
        if (alive) setError(String(e));
      }
      if (alive) timer = window.setTimeout(run, 2000);
    };
    run();
    return () => {
      alive = false;
      clearTimeout(timer);
    };
  }, []);
  const act = async (fn: () => Promise<unknown>, label: string) => {
    try {
      const r = await fn();
      push(label, "ok", { description: typeof r === "string" ? r : undefined });
    } catch (e) {
      push(String(e), "error");
    }
  };
  return (
    <section className="flex flex-col gap-3">
      <h3 className="flex items-center gap-2 text-sm">
        <Cpu size={16} />
        Runtime и профилирование
      </h3>
      {error && <p className="text-xs text-danger">{error}</p>}
      <div className="grid grid-cols-3 gap-2">
        {[
          ["Heap", formatBytes(stats?.heapBytes ?? 0)],
          ["Память Go", formatBytes(stats?.systemBytes ?? 0)],
          ["Goroutines", stats?.goroutines ?? 0],
          ["Heap objects", stats?.heapObjects ?? 0],
          ["GC cycles", stats?.gcCount ?? 0],
          ["GC pause", `${(stats?.gcPauseMs ?? 0).toFixed(1)} ms`],
        ].map(([k, v]) => (
          <div
            key={k}
            className="rounded-lg border border-border bg-surface p-3"
          >
            <div className="text-[11px] text-text-faint">{k}</div>
            <div className="font-mono text-sm">{v}</div>
          </div>
        ))}
      </div>
      <p className="text-xs text-text-muted">
        {stats?.goVersion} · PID {stats?.pid} · {stats?.cpus} CPU ·{" "}
        {formatUptime(
          Date.now() - (stats?.uptimeSeconds ?? 0) * 1000,
          Date.now(),
        )}
      </p>
      <div className="flex flex-wrap gap-2">
        <button
          className={button}
          onClick={() => act(CollectGarbage, "GC выполнен")}
        >
          <RefreshCw size={13} />
          Принудительный GC
        </button>
        <button
          className={button}
          onClick={() => act(ExportHeapProfile, "Heap profile сохранён")}
        >
          <Download size={13} />
          Heap pprof
        </button>
        <button
          className={button}
          disabled={hide}
          onClick={async () => {
            try {
              setDump(await GetGoroutineDump());
            } catch (e) {
              push(String(e), "error");
            }
          }}
        >
          Goroutine dump
        </button>
      </div>
      {dump && !hide && (
        <>
          <button
            className={button}
            onClick={() =>
              act(() => navigator.clipboard.writeText(dump), "Dump скопирован")
            }
          >
            Копировать dump
          </button>
          <pre className="max-h-72 overflow-auto rounded-lg border border-border bg-bg p-3 text-[10px]">
            {dump}
          </pre>
        </>
      )}
    </section>
  );
}
