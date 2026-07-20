import { useCallback, useEffect, useMemo, useState } from "react";
import TitleBar from "./components/TitleBar";
import Sidebar from "./components/Sidebar";
import ConnectionView from "./components/ConnectionView";
import ProfilesView from "./components/ProfilesView";
import SettingsView from "./components/SettingsView";
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

const emptyStatus: Status = {
  state: "disconnected",
  core: "sing-box",
  activeProfile: null,
  stats: { upload: 0, download: 0, uploadSpeed: 0, downloadSpeed: 0 },
  connectedAt: 0,
};

export default function App() {
  const [view, setView] = useState<ViewKey>("connection");
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([]);
  const [appInfo, setAppInfo] = useState<AppInfo | null>(null);
  const [settings, setSettings] = useState<AppSettings>({
    core: "sing-box",
    activeProfileId: "",
    routingMode: "rules",
    autoConnect: false,
    dns: "1.1.1.1",
  });
  const [status, setStatus] = useState<Status>(emptyStatus);

  // Initial load + subscribe to live status pushes from the engine.
  useEffect(() => {
    GetProfiles().then((p) => setProfiles((p as Profile[]) ?? []));
    GetSubscriptions().then((s) => setSubscriptions((s as Subscription[]) ?? []));
    GetSettings().then((s) => setSettings(s as AppSettings));
    GetStatus().then((s) => setStatus(s as Status));
    GetAppInfo().then((i) => setAppInfo(i as AppInfo));

    const off = EventsOn("vpn:status", (s: Status) => setStatus(s));
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
      await AddSubscription(name, url);
      await refreshProfiles();
    },
    [refreshProfiles]
  );

  const handleUpdateSub = useCallback(
    async (id: string) => {
      await UpdateSubscription(id);
      await refreshProfiles();
    },
    [refreshProfiles]
  );

  const handleDeleteSub = useCallback(
    async (id: string) => {
      await DeleteSubscription(id);
      await refreshProfiles();
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

  return (
    <div className="flex h-screen flex-col bg-bg text-text">
      <TitleBar />
      <div className="flex min-h-0 flex-1">
        <Sidebar active={view} onSelect={setView} />
        <main className="min-w-0 flex-1">
          {view === "connection" && (
            <ConnectionView
              status={status}
              settings={settings}
              activeProfile={activeProfile}
              onConnect={handleConnect}
              onDisconnect={handleDisconnect}
            />
          )}
          {view === "profiles" && (
            <ProfilesView
              profiles={profiles}
              subscriptions={subscriptions}
              activeId={settings.activeProfileId}
              connected={connected}
              onImportLink={handleImport}
              onAddSub={handleAddSub}
              onUpdateSub={handleUpdateSub}
              onDeleteSub={handleDeleteSub}
              onDelete={handleDelete}
              onActivate={handleActivate}
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
        </main>
      </div>
    </div>
  );
}
