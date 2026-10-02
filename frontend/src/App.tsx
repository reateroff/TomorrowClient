import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import TitleBar from "./components/TitleBar";
import Sidebar from "./components/Sidebar";
import ConnectionView from "./components/ConnectionView";
import ProfilesView from "./components/ProfilesView";
import ConfigsView from "./components/ConfigsView";
import RoutingView from "./components/RoutingView";
import SettingsView from "./components/SettingsView";
import Toasts, { push, track } from "./components/Toasts";
import type {
  AppInfo,
  AppSettings,
  Profile,
  Status,
  Subscription,
  ViewKey,
} from "./types";
import {
  Connect,
  Disconnect,
  GetProfiles,
  GetSettings,
  GetStatus,
  GetSubscriptions,
  GetAppInfo,
  ImportLinks,
  AddSubscription,
  UpdateSubscription,
  DeleteSubscription,
  DeleteProfile,
  SaveSettings,
} from "../wailsjs/go/main/App";
import { EventsOn } from "../wailsjs/runtime/runtime";
import { applyTheme } from "./theme";

const emptyStatus: Status = {
  state: "disconnected",
  core: "sing-box",
  activeProfile: null,
  stats: { upload: 0, download: 0, uploadSpeed: 0, downloadSpeed: 0 },
  connectedAt: 0,
};

const defaultSettings: AppSettings = {
  core: "xray",
  activeProfileId: "",
  dns: "1.1.1.1",
  dnsFallback: "8.8.8.8",
  tunName: "TomorrowTun",
  stack: "",
  mtu: 0,
  rules: [],
  routingMode: "simple",
  simpleFinal: "proxy",
  graph: { nodes: [], final: "proxy", layout: {} },
  ipv6: false,
  strictRoute: true,
  sniff: true,
  hwidEnabled: true,
  hwid: "",
  deviceOs: "",
  osVersion: "",
  deviceModel: "",
  userAgent: "",
  pingMethod: "get",
  pingUrl: "http://cp.cloudflare.com/generate_204",
  pingTimeout: 5000,
  autoConnect: false,
  launchAtStartup: false,
  minimizeToTray: false,
  devMode: false,
  demoMode: false,
  theme: "smoke",
  accent: "sunset-mist",
  savedColors: [],
  font: "rubik",
  radius: "soft",
  navPosition: "top",
  animation: "fade",
};

export default function App() {
  const [view, setView] = useState<ViewKey>("connection");
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([]);
  const [appInfo, setAppInfo] = useState<AppInfo | null>(null);
  const [settings, setSettings] = useState<AppSettings>(defaultSettings);
  const [status, setStatus] = useState<Status>(emptyStatus);
  const pendingSettings=useRef<AppSettings|null>(null),settingsTimer=useRef<ReturnType<typeof setTimeout>|null>(null),saveQueue=useRef<Promise<unknown>>(Promise.resolve());
  const persistSettings=useCallback((next:AppSettings)=>{pendingSettings.current=next;if(settingsTimer.current)clearTimeout(settingsTimer.current);settingsTimer.current=setTimeout(()=>{const latest=pendingSettings.current;pendingSettings.current=null;if(latest)saveQueue.current=saveQueue.current.catch(()=>{}).then(()=>SaveSettings(latest as any)).catch(e=>push(`Не удалось сохранить настройки: ${String(e)}`,"error"))},100)},[]);
  const flushSettings=useCallback(async()=>{if(settingsTimer.current)clearTimeout(settingsTimer.current);const latest=pendingSettings.current;pendingSettings.current=null;if(latest)saveQueue.current=saveQueue.current.catch(()=>{}).then(()=>SaveSettings(latest as any));await saveQueue.current},[]);
  // Which profile group is open in the Configs tab ("manual" or a sub id).
  const [selectedGroup, setSelectedGroup] = useState<string>("");

  // Initial load + subscribe to live status pushes from the engine.
  useEffect(() => {
    GetProfiles().then((p) => setProfiles((p as Profile[]) ?? []));
    GetSubscriptions().then((s) => setSubscriptions((s as Subscription[]) ?? []));
    GetSettings().then((s) => {
      const cfg = s as AppSettings;
      setSettings(cfg);
      applyTheme(cfg);
    });
    GetStatus().then((s) => setStatus(s as Status));
    GetAppInfo().then((i) => setAppInfo(i as AppInfo));

    let prev: Status["state"] | null = null;
    const off = EventsOn("vpn:status", (s: Status) => {
      setStatus(s);
      // Only surface errors — routine connect/disconnect is visible in the UI.
      if (s.state !== prev) {
        if (s.state === "error")
          push("Не удалось подключиться", "error", { description: s.error || undefined });
        prev = s.state;
      }
    });

    // A factory reset wipes the store behind our back, so reload everything
    // instead of leaving deleted servers on screen.
    const offReset = EventsOn("app:datareset", () => {
      if(settingsTimer.current)clearTimeout(settingsTimer.current);pendingSettings.current=null;
      setProfiles([]);
      setSubscriptions([]);
      setSelectedGroup("");
      GetSettings().then((s) => {
        const cfg = s as AppSettings;
        setSettings(cfg);
        applyTheme(cfg);
      });
      GetStatus().then((s) => setStatus(s as Status));
      push("Все данные удалены", "info");
    });

    return () => {
      off();
      offReset();
    };
  }, []);

  const refreshProfiles = useCallback(async () => {
    const [p, s] = await Promise.all([GetProfiles(), GetSubscriptions()]);
    setProfiles((p as Profile[]) ?? []);
    setSubscriptions((s as Subscription[]) ?? []);
  }, []);

  const activeProfile = useMemo(
    () => profiles.find((p) => p.id === settings.activeProfileId) ?? null,
    [profiles, settings.activeProfileId]
  );

  const connected = status.state === "connected";

  const handleImport = useCallback(
    async (raw: string) => {
      await flushSettings();
      const added = ((await ImportLinks(raw)) as Profile[]) ?? [];
      await refreshProfiles();
      if (added.length > 1) push(`Добавлено серверов: ${added.length}`, "ok");
      // First imported server becomes active automatically.
      if (!settings.activeProfileId && added.length > 0) {
        const next = { ...settings, activeProfileId: added[0].id };
        setSettings(next);
        await SaveSettings(next as any);
      }
    },
    [refreshProfiles, settings,flushSettings]
  );

  const handleDelete = useCallback(
    async (id: string) => {
      if(settings.activeProfileId===id && status.state!=="disconnected" && status.state!=="error")throw new Error("Сначала отключите активное соединение");
      await flushSettings();
      await DeleteProfile(id);
      await refreshProfiles();
      if(settings.activeProfileId===id){const next={...settings,activeProfileId:""};setSettings(next);await SaveSettings(next as any)}
    },
    [refreshProfiles,settings,status.state,flushSettings]
  );

  const handleAddSub = useCallback(
    async (name: string, url: string) => {
      try {
        const s = (await AddSubscription(name, url)) as Subscription;
        await refreshProfiles();
        push(`Подписка «${s.name}» добавлена: ${s.count} серв.`, "ok");
      } catch (e) {
        push(`Не удалось добавить подписку: ${String(e)}`, "error");
        throw e;
      }
    },
    [refreshProfiles]
  );

  const handleUpdateSub = useCallback(
    async (id: string) => {
      try {
        await track(UpdateSubscription(id) as Promise<Subscription>, {
          loading: "Обновляю подписку…",
          success: (s) => `Подписка «${s.name}» обновлена: ${s.count} серв.`,
          error: (e) => `Ошибка обновления: ${String(e)}`,
        });
        await refreshProfiles();
      } catch {
        /* the toast already reported it */
      }
    },
    [refreshProfiles]
  );

  const handleDeleteSub = useCallback(
    async (id: string) => {
      await DeleteSubscription(id);
      await refreshProfiles();
      push("Подписка удалена", "info");
    },
    [refreshProfiles]
  );

  const handleActivate = useCallback(
    async (id: string) => {
      await flushSettings();
      const next = { ...settings, activeProfileId: id };
      setSettings(next);
      await SaveSettings(next as any);
    },
    [settings,flushSettings]
  );

  const handleSettings = useCallback((next: AppSettings) => {
    setSettings(next);
    applyTheme(next);
    persistSettings(next);
  }, [persistSettings]);

  const handleConnect = useCallback(() => {
    flushSettings().then(()=>Connect(settings.activeProfileId)).catch(() => {
      /* errors surface via the status event */
    });
  }, [settings.activeProfileId,flushSettings]);

  const handleDisconnect = useCallback(() => {
    Disconnect();
  }, []);

  const navPosition=(["left","top","right","bottom"].includes(settings.navPosition)?settings.navPosition:"left") as "left"|"top"|"right"|"bottom";

  return (
    <div className="relative flex h-screen flex-col bg-bg text-text">
      <TitleBar minimizeToTray={settings.minimizeToTray} />
      {view === "routing" && settings.routingMode === "pro" && (
        <div className="window-resize-frame" aria-hidden="true">
          {(["top", "right", "bottom", "left"] as const).map(edge => (
            <div key={edge} className={`window-resize-edge window-resize-${edge}`} />
          ))}
        </div>
      )}
      <div className={`flex min-h-0 flex-1 ${navPosition === "top" ? "flex-col" : navPosition === "bottom" ? "flex-col-reverse" : navPosition === "right" ? "flex-row-reverse" : ""}`}>
        <Sidebar
          active={view}
          position={navPosition}
          onSelect={setView}
        />
        <main className="relative min-h-0 min-w-0 flex-1 overflow-hidden">
          <div key={view} className={`h-full min-h-0 ${view === "settings" ? "" : "animate-view"}`} >
          {view === "connection" && (
            <ConnectionView
              status={status}
              activeProfile={activeProfile}
              onConnect={handleConnect}
              onDisconnect={handleDisconnect}
            />
          )}
          {view === "profiles" && (
            <ProfilesView
              profiles={profiles}
              subscriptions={subscriptions}
              selectedGroup={selectedGroup}
              hideData={settings.demoMode}
              onImportLink={handleImport}
              onAddSub={handleAddSub}
              onUpdateSub={handleUpdateSub}
              onDeleteSub={handleDeleteSub}
              onDeleteProfile={handleDelete}
              onSelectGroup={setSelectedGroup}
            />
          )}
          {view === "configs" && (
            <ConfigsView
              profiles={profiles}
              subscriptions={subscriptions}
              selectedGroup={selectedGroup}
              activeId={settings.activeProfileId}
              connected={connected}
              hideData={settings.demoMode}
              core={settings.core}
              onActivate={handleActivate}
              onChanged={refreshProfiles}
              onDelete={handleDelete}
            />
          )}
          {view === "routing" && (
            <RoutingView
              settings={settings}
              disabled={connected || status.state === "connecting"}
              onChange={handleSettings}
            />
          )}
          {view === "settings" && (
            <SettingsView
              settings={settings}
              appInfo={appInfo}
              disabled={connected || status.state === "connecting"}
              onChange={handleSettings}
            />
          )}
          </div>
        <Toasts lifted={view === "routing" && settings.routingMode === "pro"} />
      </main>
      </div>
    </div>
  );
}
