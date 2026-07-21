import { useCallback, useEffect, useMemo, useState } from "react";
import TitleBar from "./components/TitleBar";
import Sidebar from "./components/Sidebar";
import ConnectionView from "./components/ConnectionView";
import ProfilesView from "./components/ProfilesView";
import ConfigsView from "./components/ConfigsView";
import RoutingView from "./components/RoutingView";
import SettingsView from "./components/SettingsView";
import Toasts, { push } from "./components/Toasts";
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
  ImportLink,
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
  core: "sing-box",
  activeProfileId: "",
  routingMode: "rules",
  dns: "1.1.1.1",
  tunName: "TomorrowTun",
  stack: "gvisor",
  mtu: 0,
  rules: [],
  autoConnect: false,
  launchAtStartup: false,
  minimizeToTray: false,
  theme: "graphite",
  accent: "indigo",
  font: "inter",
  radius: "soft",
  navPosition: "left",
};

export default function App() {
  const [view, setView] = useState<ViewKey>("connection");
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([]);
  const [appInfo, setAppInfo] = useState<AppInfo | null>(null);
  const [settings, setSettings] = useState<AppSettings>(defaultSettings);
  const [status, setStatus] = useState<Status>(emptyStatus);
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
          push(s.error ? `Ошибка: ${s.error}` : "Ошибка подключения", "error");
        prev = s.state;
      }
    });
    return () => off();
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
      const p = (await ImportLink(raw)) as Profile;
      await refreshProfiles();
      // First imported server becomes active automatically.
      if (!settings.activeProfileId) {
        const next = { ...settings, activeProfileId: p.id };
        setSettings(next);
        await SaveSettings(next as any);
      }
    },
    [refreshProfiles, settings]
  );

  const handleDelete = useCallback(
    async (id: string) => {
      await DeleteProfile(id);
      await refreshProfiles();
    },
    [refreshProfiles]
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
        const s = (await UpdateSubscription(id)) as Subscription;
        await refreshProfiles();
        push(`Подписка «${s.name}» обновлена: ${s.count} серв.`, "ok");
      } catch (e) {
        push(`Ошибка обновления: ${String(e)}`, "error");
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
      const next = { ...settings, activeProfileId: id };
      setSettings(next);
      await SaveSettings(next as any);
    },
    [settings]
  );

  const handleSettings = useCallback(async (next: AppSettings) => {
    setSettings(next);
    applyTheme(next);
    await SaveSettings(next as any);
  }, []);

  const handleConnect = useCallback(() => {
    Connect(settings.activeProfileId).catch(() => {
      /* errors surface via the status event */
    });
  }, [settings.activeProfileId]);

  const handleDisconnect = useCallback(() => {
    Disconnect();
  }, []);

  const navTop = settings.navPosition === "top";

  return (
    <div className="relative flex h-screen flex-col bg-bg text-text">
      <TitleBar minimizeToTray={settings.minimizeToTray} />
      <div className={`flex min-h-0 flex-1 ${navTop ? "flex-col" : ""}`}>
        <Sidebar
          active={view}
          position={navTop ? "top" : "left"}
          onSelect={setView}
        />
        <main className="relative min-h-0 min-w-0 flex-1 overflow-hidden">
          {view === "connection" && (
            <ConnectionView
              status={status}
              settings={settings}
              activeProfile={activeProfile}
              onConnect={handleConnect}
              onDisconnect={handleDisconnect}
              onOpenConfigs={() => {
                if (activeProfile)
                  setSelectedGroup(activeProfile.subId || "manual");
                setView("configs");
              }}
            />
          )}
          {view === "profiles" && (
            <ProfilesView
              profiles={profiles}
              subscriptions={subscriptions}
              selectedGroup={selectedGroup}
              onImportLink={handleImport}
              onAddSub={handleAddSub}
              onUpdateSub={handleUpdateSub}
              onDeleteSub={handleDeleteSub}
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
              onActivate={handleActivate}
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
        <Toasts />
      </main>
      </div>
    </div>
  );
}
