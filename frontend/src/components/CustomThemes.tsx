import { useState } from "react";
import { Plus, Pencil, Trash2, Copy, X, Check, Palette } from "lucide-react";
import { THEME_PRESETS, ACCENTS, isHex, contrastText } from "../theme";
import type { AppSettings, CustomTheme } from "../types";
import { push } from "./Toasts";
import ModalPortal from "./ModalPortal";
import ColorPickerModal from "./ColorPickerModal";
import Menu from "./Menu";
import SegmentedControl from "./SegmentedControl";
import { confirmAction } from "./ClientConfirm";
const fields = [
  ["bg", "Фон"],
  ["surface", "Поверхности"],
  ["surface2", "Карточки и меню"],
  ["border", "Границы"],
  ["borderSoft", "Мягкие границы"],
  ["text", "Текст"],
  ["textMuted", "Вторичный текст"],
  ["textFaint", "Подписи"],
  ["accent", "Акцент"],
  ["success", "Успех"],
  ["danger", "Ошибка"],
];
export default function CustomThemes({
  settings,
  set,
}: {
  settings: AppSettings;
  set: (p: Partial<AppSettings>) => void;
}) {
  const themes = settings.customThemes ?? [];
  const [mode,setMode]=useState("easy");
  const simpleFields=fields.filter(([k])=>["bg","surface","text","accent","success","danger"].includes(k));
  const mix=(a:string,b:string,t:number)=>"#"+[0,2,4].map(i=>Math.round(parseInt(a.slice(i+1,i+3),16)*(1-t)+parseInt(b.slice(i+1,i+3),16)*t).toString(16).padStart(2,"0")).join("");
  const updateColor=(key:string,hex:string)=>{if(!draft)return;const colors={...draft.colors};if(key!=="accent")colors[key]=hex;if(mode==="easy"){const bg=colors.bg,text=colors.text;colors.surface2=mix(colors.surface,text,0.055);colors.border=mix(bg,text,0.16);colors.borderSoft=mix(bg,text,0.095);colors.textMuted=mix(bg,text,0.62);colors.textFaint=mix(bg,text,0.43)}setDraft({...draft,colors,accent:key==="accent"?hex:draft.accent})};
  const [draft, setDraft] = useState<CustomTheme | null>(null),
    [colorKey, setColorKey] = useState<string | null>(null);
  const base = (): CustomTheme => {
    const c =
      themes.find((t) => t.id === settings.theme) ??
      THEME_PRESETS.find((t) => t.id === settings.theme) ??
      THEME_PRESETS[0];
    return {
      id: `custom-${crypto.randomUUID()}`,
      name: "Моя тема",
      colors: { success: "#5bd6a0", danger: "#ff6b6b", ...c.colors },
      accent: isHex(settings.accent)
        ? settings.accent
        : (ACCENTS.find((a) => a.id === settings.accent)?.color ?? "#7c8cff"),
    };
  };
  const save = () => {
    if (
      !draft ||
      !draft.name.trim() ||
      !fields.every(([k]) =>
        isHex(k === "accent" ? draft.accent : (draft.colors[k] ?? "")),
      )
    ) {
      push("Укажите название и корректные цвета", "error");
      return;
    }
    set({
      customThemes: [
        ...themes.filter((t) => t.id !== draft.id),
        { ...draft, name: draft.name.trim() },
      ],
      theme: draft.id,
      accent: draft.accent,
    });
    setDraft(null);
    push("Тема сохранена", "ok");
  };
  const remove = async (t: CustomTheme) => {
    if (
      !(await confirmAction({
        title: "Удалить тему?",
        description: `Тема «${t.name}» будет удалена из сохранённых.`,
        confirmLabel: "Удалить",
        danger: true,
      }))
    )
      return;
    set({
      customThemes: themes.filter((x) => x.id !== t.id),
      ...(settings.theme === t.id ? { theme: "graphite" } : {}),
    });
  };
  return (
    <div className="mt-3">
      {themes.length > 0 && (
        <div className="mb-3 grid grid-cols-2 gap-2">
          {themes.map((t) => (
            <div
              key={t.id}
              className={`group flex min-w-0 items-center gap-2 rounded-lg border px-2 py-1.5 ${settings.theme === t.id ? "border-accent/50 bg-surface-2" : "border-border bg-surface"}`}
            >
              <button
                className="flex min-w-0 flex-1 items-center gap-2 p-1 text-left text-xs text-text-muted hover:text-text"
                onClick={() => set({ theme: t.id, accent: t.accent })}
              >
                <span
                  className="grid h-6 w-6 shrink-0 place-items-center rounded-md border"
                  style={{
                    background: t.colors.bg,
                    borderColor: t.colors.border,
                  }}
                >
                  <span
                    className="h-2 w-2 rounded-full"
                    style={{ background: t.accent }}
                  />
                </span>
                <span className="truncate">{t.name}</span>
                {settings.theme === t.id && (
                  <Check size={12} className="ml-auto shrink-0 text-accent" />
                )}
              </button>
              <Menu
                label={`Меню темы ${t.name}`}
                items={[
                  {
                    label: "Редактировать",
                    icon: <Pencil size={13} />,
                    onClick: () => {setMode("pro");setDraft({ ...t, colors: { ...t.colors } })},
                  },
                  {
                    label: "Создать копию",
                    icon: <Copy size={13} />,
                    onClick: () =>
                      setDraft({
                        ...t,
                        id: `custom-${crypto.randomUUID()}`,
                        name: t.name + " (копия)",
                        colors: { ...t.colors },
                      }),
                  },
                  {
                    label: "Удалить",
                    icon: <Trash2 size={13} />,
                    danger: true,
                    onClick: () => {
                      void remove(t);
                    },
                  },
                ]}
              />
            </div>
          ))}
        </div>
      )}
      <button
        onClick={() => {setMode("easy");setDraft(base())}}
        className="no-drag inline-flex items-center gap-2.5 rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text-muted transition hover:bg-surface-2 hover:text-text"
      >
        <span
          className="grid h-5 w-5 place-items-center rounded-full border border-border"
          style={{
            background:
              "conic-gradient(from 0deg,#9da8cb,#a6bbb0,#c0b0bc,#9da8cb)",
          }}
        >
          <Plus size={12} className="text-bg" />
        </span>
        Создать свою тему
      </button>
      {draft && (
        <ModalPortal
          label="Редактор темы"
          onClose={() => {
            setColorKey(null);
            setDraft(null);
          }}
        >
          <div
            className="fixed inset-0 z-50 grid place-items-center bg-black/60 p-6 backdrop-blur-sm"
            onClick={() => setDraft(null)}
          >
            <div
              className="animate-view flex max-h-full w-full max-w-lg flex-col overflow-hidden rounded-xl border border-border bg-surface shadow-2xl"
              onClick={(e) => e.stopPropagation()}
            >
              <div className="flex shrink-0 items-center gap-2.5 border-b border-border px-4 py-3">
                <Palette size={16} className="text-accent" />
                <h2 className="flex-1 text-sm font-medium">Своя тема</h2>
                <button
                  aria-label="Закрыть редактор темы"
                  onClick={() => setDraft(null)}
                  className="text-text-faint hover:text-text"
                >
                  <X size={16} />
                </button>
              </div>
              <div className="min-h-0 overflow-y-auto p-3">
                <input
                  aria-label="Название темы"
                  maxLength={48}
                  value={draft.name}
                  onChange={(e) => setDraft({ ...draft, name: e.target.value })}
                  className="mb-2 w-full rounded-lg border border-border bg-bg px-3 py-2 text-sm text-text outline-none focus:border-accent/60"
                  placeholder="Название темы"
                />
                <div
                  className="mb-2 flex items-center gap-3 rounded-lg border p-2.5"
                  style={{
                    background: draft.colors.bg,
                    borderColor: draft.colors.border,
                    color: draft.colors.text,
                  }}
                >
                  <span
                    className="grid h-8 w-8 place-items-center rounded-md"
                    style={{
                      background: draft.colors.surface2,
                      color: draft.accent,
                    }}
                  >
                    <Palette size={16} />
                  </span>
                  <div className="min-w-0 flex-1">
                    <div className="text-xs">{draft.name || "Моя тема"}</div>
                    <div
                      className="mt-0.5 text-[11px]"
                      style={{ color: draft.colors.textMuted }}
                    >
                      Предпросмотр оформления
                    </div>
                  </div>
                  <span
                    className="rounded-md px-3 py-1.5 text-[11px]"
                    style={{
                      background: draft.accent,
                      color: contrastText(draft.accent),
                    }}
                  >
                    Подключить
                  </span>
                </div>
                <div className="mb-2 flex items-center justify-between gap-3"><span className="text-[11px] text-text-faint">{mode==="easy"?"Основные цвета · оттенки автоматически":"Все цвета интерфейса отдельно"}</span><SegmentedControl value={mode} onChange={setMode} options={[{id:"easy",label:"Простой"},{id:"pro",label:"Pro"}]}/></div>
                <div className="grid grid-cols-2 gap-x-3 gap-y-1">
                  {(mode==="easy"?simpleFields:fields).map(([k, label]) => {
                    const color =
                      k === "accent"
                        ? draft.accent
                        : (draft.colors[k] ?? "#808080");
                    return (
                      <button
                        key={k}
                        aria-label={`Цвет темы: ${label}`}
                        onClick={() => setColorKey(k)}
                        className="flex min-w-0 items-center gap-2 rounded-md px-2 py-1 text-left text-xs text-text-muted transition hover:bg-surface-2"
                      >
                        <span
                          className="h-5 w-5 shrink-0 rounded-full border border-border"
                          style={{ background: color }}
                        />
                        <span className="min-w-0 flex-1 truncate">{label}</span>
                        <span className="hidden font-mono text-[10px] text-text-faint min-[720px]:inline">
                          {color}
                        </span>
                      </button>
                    );
                  })}
                </div>
              </div>
              <div className="flex shrink-0 justify-end gap-2 border-t border-border px-4 py-3">
                <button
                  onClick={() => setDraft(null)}
                  className="rounded-lg border border-border px-4 py-2 text-xs text-text-muted transition hover:bg-surface-2"
                >
                  Отмена
                </button>
                <button
                  onClick={save}
                  className="rounded-lg bg-accent px-4 py-2 text-xs font-medium text-on-accent transition hover:bg-accent-soft"
                >
                  Сохранить тему
                </button>
              </div>
            </div>
          </div>
        </ModalPortal>
      )}
      {draft && colorKey && (
        <ColorPickerModal
          title={fields.find(([k]) => k === colorKey)?.[1] || "Цвет темы"}
          initial={
            colorKey === "accent"
              ? draft.accent
              : (draft.colors[colorKey] ?? "#808080")
          }
          saved={settings.savedColors ?? []}
          onClose={() => setColorKey(null)}
          onApply={(hex) => {
            updateColor(colorKey,hex);
            setColorKey(null);
          }}
          onSave={(hex) =>
            set({
              savedColors: [
                ...new Set([
                  ...(settings.savedColors ?? []),
                  hex.toLowerCase(),
                ]),
              ].slice(-16),
            })
          }
          onRemove={(hex) =>
            set({
              savedColors: (settings.savedColors ?? []).filter(
                (c) => c !== hex.toLowerCase(),
              ),
            })
          }
        />
      )}
    </div>
  );
}
