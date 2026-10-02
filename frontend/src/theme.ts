// Appearance theming. Presets, accent palette, fonts and corner rounding are
// defined here and applied to the document root as CSS custom properties so the
// whole UI (which reads var(--color-*) / var(--radius-lg) / var(--font-sans))
// updates live without a reload.

export interface ThemePreset {
  id: string;
  name: string;
  colors: {
    bg: string;
    surface: string;
    surface2: string;
    border: string;
    borderSoft: string;
    text: string;
    textMuted: string;
    textFaint: string;
  };
}

// How many presets the Appearance screen shows before "показать ещё".
export const THEMES_VISIBLE = 6;

// Dark presets, no harsh pure white text. "smoke" is the shipped default and
// the first THEMES_VISIBLE entries are the ones shown unexpanded, so order
// matters: keep the plainest six first.
export const THEME_PRESETS: ThemePreset[] = [
  {
    id: "graphite",
    name: "Графит",
    colors: {
      bg: "#0e0e11",
      surface: "#16161a",
      surface2: "#1c1c21",
      border: "#26262c",
      borderSoft: "#202026",
      text: "#e7e7ea",
      textMuted: "#9a9aa4",
      textFaint: "#6b6b76",
    },
  },
  {
    id: "midnight",
    name: "Полночь",
    colors: {
      bg: "#0b0d12",
      surface: "#12141b",
      surface2: "#191c25",
      border: "#242835",
      borderSoft: "#1d212b",
      text: "#e6e8ef",
      textMuted: "#949aab",
      textFaint: "#666d80",
    },
  },
  {
    id: "coal",
    name: "Уголь",
    colors: {
      bg: "#0c0c0c",
      surface: "#141414",
      surface2: "#1b1b1b",
      border: "#282828",
      borderSoft: "#1f1f1f",
      text: "#eaeaea",
      textMuted: "#9a9a9a",
      textFaint: "#6a6a6a",
    },
  },
  {
    id: "obsidian",
    name: "Обсидиан",
    colors: {
      bg: "#000000",
      surface: "#0b0b0d",
      surface2: "#141417",
      border: "#222226",
      borderSoft: "#181819",
      text: "#ededf0",
      textMuted: "#98989f",
      textFaint: "#616168",
    },
  },
  {
    id: "espresso",
    name: "Эспрессо",
    colors: {
      bg: "#100d0b",
      surface: "#181310",
      surface2: "#211a16",
      border: "#2e2620",
      borderSoft: "#241d18",
      text: "#ece5df",
      textMuted: "#a89c92",
      textFaint: "#75695f",
    },
  },
  {
    id: "nord",
    name: "Норд",
    colors: {
      bg: "#0d1017",
      surface: "#141924",
      surface2: "#1c2230",
      border: "#2a3242",
      borderSoft: "#212836",
      text: "#e5e9f0",
      textMuted: "#94a0b5",
      textFaint: "#647089",
    },
  },
  // --- Revealed by "показать ещё" ---
  {
    id: "plum",
    name: "Слива",
    colors: {
      bg: "#100c14",
      surface: "#18121d",
      surface2: "#211926",
      border: "#2f2436",
      borderSoft: "#251c2b",
      text: "#eae3ee",
      textMuted: "#a397ab",
      textFaint: "#726579",
    },
  },
  {
    id: "ocean",
    name: "Океан",
    colors: {
      bg: "#081114",
      surface: "#0e1a1f",
      surface2: "#15242b",
      border: "#21353d",
      borderSoft: "#1a2b32",
      text: "#e2eef1",
      textMuted: "#8fa6ad",
      textFaint: "#5f767e",
    },
  },
  {
    id: "moss",
    name: "Мох",
    colors: {
      bg: "#0c110d",
      surface: "#121a14",
      surface2: "#1a241c",
      border: "#26332a",
      borderSoft: "#1e2921",
      text: "#e4ebe4",
      textMuted: "#97a598",
      textFaint: "#67756a",
    },
  },
  {
    id: "wine",
    name: "Вино",
    colors: {
      bg: "#120c0e",
      surface: "#1b1214",
      surface2: "#241a1c",
      border: "#342528",
      borderSoft: "#291d20",
      text: "#eee3e5",
      textMuted: "#ab979a",
      textFaint: "#7a6568",
    },
  },
  {
    id: "steel",
    name: "Сталь",
    colors: {
      bg: "#0f1216",
      surface: "#171b21",
      surface2: "#1f242c",
      border: "#2d343e",
      borderSoft: "#242a33",
      text: "#e6e9ee",
      textMuted: "#99a1ae",
      textFaint: "#69727f",
    },
  },
  {
    id: "smoke",
    name: "Дым",
    colors: {
      bg: "#111010",
      surface: "#1a1818",
      surface2: "#232020",
      border: "#322e2e",
      borderSoft: "#282525",
      text: "#eae7e5",
      textMuted: "#a49f9c",
      textFaint: "#736e6b",
    },
  },
];

// Soft-tinted additions; surfaces stay quiet and borders are not accents.
THEME_PRESETS.push(...[{"id": "aurora", "name": "Аврора", "colors": {"bg": "#101219", "surface": "#181c27", "surface2": "#202637", "border": "#343d52", "borderSoft": "#292f40", "text": "#e6e9f1", "textMuted": "#a0a8bd", "textFaint": "#747e96"}}, {"id": "neon", "name": "Неон", "colors": {"bg": "#131018", "surface": "#1d1825", "surface2": "#282131", "border": "#3e334b", "borderSoft": "#302838", "text": "#ebe5ef", "textMuted": "#afa1bb", "textFaint": "#807389"}}, {"id": "lagoon", "name": "Лагуна", "colors": {"bg": "#0e1518", "surface": "#172126", "surface2": "#202d33", "border": "#34464e", "borderSoft": "#29383f", "text": "#e3ecee", "textMuted": "#9cafb5", "textFaint": "#72878f"}}, {"id": "sunset", "name": "Закат", "colors": {"bg": "#181214", "surface": "#251b1e", "surface2": "#302328", "border": "#4a363e", "borderSoft": "#392a30", "text": "#eee5e5", "textMuted": "#b7a2a4", "textFaint": "#8a757a"}}, {"id": "mint-light", "name": "Светлая мята", "colors": {"bg": "#f2f5f3", "surface": "#fafcfb", "surface2": "#e8eeeb", "border": "#cad6d0", "borderSoft": "#dbe4df", "text": "#263d32", "textMuted": "#546d60", "textFaint": "#708778"}}, {"id": "sky-light", "name": "Светлое небо", "colors": {"bg": "#f2f4f7", "surface": "#fafbfd", "surface2": "#e8edf4", "border": "#c9d2df", "borderSoft": "#dce2eb", "text": "#293b53", "textMuted": "#576b86", "textFaint": "#7588a1"}}]);

export interface Accent {
  id: string;
  name: string;
  color: string;
  soft: string;
}

// Accent palette — bright staples plus a few muted/dark tones. Users can also
// enter a custom hex (handled in applyTheme).
export const ACCENTS: Accent[] = [
  { id: "indigo", name: "Индиго", color: "#7c8cff", soft: "#a5b0ff" },
  { id: "blue", name: "Синий", color: "#4f8cff", soft: "#83adff" },
  { id: "sky", name: "Небо", color: "#38bdf8", soft: "#7dd3fc" },
  { id: "teal", name: "Бирюза", color: "#4fd1c5", soft: "#7ee0d7" },
  { id: "emerald", name: "Изумруд", color: "#5bd6a0", soft: "#84e4bb" },
  { id: "lime", name: "Лайм", color: "#a3e635", soft: "#c4f06f" },
  { id: "amber", name: "Янтарь", color: "#e0b155", soft: "#ecc987" },
  { id: "orange", name: "Оранж", color: "#fb923c", soft: "#fdba74" },
  { id: "rose", name: "Роза", color: "#f08a9c", soft: "#f5adba" },
  { id: "red", name: "Красный", color: "#f26d6d", soft: "#f79a9a" },
  { id: "violet", name: "Фиалка", color: "#b18cff", soft: "#c9b0ff" },
  { id: "fuchsia", name: "Фуксия", color: "#e879f9", soft: "#f0abfc" },
  { id: "slate", name: "Сталь", color: "#64748b", soft: "#94a3b8" },
  { id: "steel", name: "Свинец", color: "#5b6b8c", soft: "#8494b3" },
  // Muted / dark tones that sit quietly against dark themes.
  { id: "deep-indigo", name: "Тёмный индиго", color: "#4c4f8f", soft: "#6d70b3" },
  { id: "deep-teal", name: "Тёмная бирюза", color: "#2f7a72", soft: "#469b91" },
  { id: "deep-green", name: "Тёмный хвойный", color: "#3a7a55", soft: "#529b71" },
  { id: "deep-wine", name: "Винный", color: "#8f4550", soft: "#b1636e" },
  { id: "deep-bronze", name: "Бронза", color: "#8a6a3a", soft: "#ab8955" },
  { id: "graphite-accent", name: "Графит", color: "#5a5f6b", soft: "#7c828f" },
  {"id": "warm-sand", "name": "Дым · тёплый песок", "color": "#bca38f", "soft": "#d0bdaa"},
  {"id": "plum-mist", "name": "Слива · сирень", "color": "#a58ead", "soft": "#bca8c2"},
  {"id": "ocean-mist", "name": "Океан · морская пена", "color": "#7bafb7", "soft": "#9bc5cb"},
  {"id": "moss-mist", "name": "Мох · шалфей", "color": "#95ac8a", "soft": "#b2c3aa"},
  {"id": "wine-mist", "name": "Вино · пыльная роза", "color": "#bb8d96", "soft": "#cfa9b0"},
  {"id": "steel-mist", "name": "Сталь · серебро", "color": "#9aaabc", "soft": "#b5c1ce"},
  {"id": "aurora-mist", "name": "Аврора · лавандовый", "color": "#929fc4", "soft": "#b0b9d5"},
  {"id": "sunset-mist", "name": "Закат · персиковый", "color": "#c79985", "soft": "#d9b6a6"},
];

export interface FontChoice {
  id: string;
  name: string;
  stack: string;
}

// Font stacks; the bundled families load via @font-face in style.css.
export const FONTS: FontChoice[] = [
  { id: "inter", name: "Inter", stack: '"Inter", "Segoe UI", system-ui, sans-serif' },
  { id: "onest", name: "Onest", stack: '"Onest", "Inter", system-ui, sans-serif' },
  { id: "geist", name: "Geist", stack: '"Geist", "Inter", system-ui, sans-serif' },
  { id: "mono", name: "JetBrains Mono", stack: '"JetBrains Mono", ui-monospace, monospace' },
  { id: "manrope", name: "Manrope", stack: '"Manrope", "Onest", system-ui, sans-serif' },
  { id: "rubik", name: "Rubik", stack: '"Rubik", "Onest", system-ui, sans-serif' },
  { id: "nunito", name: "Nunito Sans", stack: '"Nunito Sans", "Onest", system-ui, sans-serif' },
  { id: "open-sans", name: "Open Sans", stack: '"Open Sans", "Onest", system-ui, sans-serif' },
];

export interface RadiusChoice {
  id: string;
  name: string;
  value: string;
}

// Corner rounding presets applied to --radius-lg.
export const RADII: RadiusChoice[] = [
  { id: "sharp", name: "Острые", value: "6px" },
  { id: "soft", name: "Мягкие", value: "14px" },
  { id: "round", name: "Круглые", value: "22px" },
];

export interface AnimationChoice {
  id: string;
  name: string;
  desc: string;
  /** CSS @keyframes name from style.css, or "none" to disable. */
  keyframes: string;
  duration: string;
}

// Entrance animation presets for views and modals. "rise" is the original
// behaviour and stays the default.
export const ANIMATIONS: AnimationChoice[] = [
  {
    id: "rise",
    name: "Подъём",
    desc: "Всплывает снизу",
    keyframes: "anim-rise",
    duration: "0.25s",
  },
  {
    id: "slide",
    name: "Слайд",
    desc: "Выезжает справа",
    keyframes: "anim-slide",
    duration: "0.28s",
  },
  {
    id: "fade",
    name: "Затухание",
    desc: "Только прозрачность",
    keyframes: "anim-fade",
    duration: "0.2s",
  },
  {
    id: "scale",
    name: "Приближение",
    desc: "Мягкий зум",
    keyframes: "anim-scale",
    duration: "0.22s",
  },
  {"id": "slide-left", "name": "Слева", "desc": "Мягкий сдвиг слева", "keyframes": "anim-slide-left", "duration": "0.28s"},
  {"id": "drop", "name": "Спуск", "desc": "Появляется сверху", "keyframes": "anim-drop", "duration": "0.28s"},
  {"id": "zoom-out", "name": "Отдаление", "desc": "Из крупного в обычный", "keyframes": "anim-zoom-out", "duration": "0.3s"},
  {"id": "soft-blur", "name": "Фокус", "desc": "Плавно набирает чёткость", "keyframes": "anim-soft-blur", "duration": "0.32s"},
  {"id": "diagonal", "name": "Диагональ", "desc": "Лёгкий подъём со сдвигом", "keyframes": "anim-diagonal", "duration": "0.3s"},
  {
    id: "none",
    name: "Без анимаций",
    desc: "Мгновенно",
    keyframes: "none",
    duration: "0s",
  },
];

function byId<T extends { id: string }>(list: T[], id: string, fallback: T): T {
  return list.find((x) => x.id === id) ?? fallback;
}

// isHex reports whether a string is a #rgb / #rrggbb colour.
export function isHex(v: string): boolean {
  return /^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/.test(v.trim());
}

export interface HSV {
  h: number; // 0..360
  s: number; // 0..1
  v: number; // 0..1
}

// The colour picker keeps HSV as its source of truth: hue survives a trip to
// pure black or pure white, which it would not if hex were authoritative.

function toHex2(n: number): string {
  return Math.round(n).toString(16).padStart(2, "0");
}

// hsvToHex converts HSV to a #rrggbb string.
export function hsvToHex(h: number, s: number, v: number): string {
  const c = v * s;
  const x = c * (1 - Math.abs(((h / 60) % 2) - 1));
  const m = v - c;
  const seg = Math.floor(((h % 360) + 360) % 360 / 60);
  const [r, g, b] = [
    [c, x, 0],
    [x, c, 0],
    [0, c, x],
    [0, x, c],
    [x, 0, c],
    [c, 0, x],
  ][seg];
  return `#${toHex2((r + m) * 255)}${toHex2((g + m) * 255)}${toHex2((b + m) * 255)}`;
}

// hexToHsv parses a #rgb / #rrggbb string, or returns null when it is not one.
export function hexToHsv(hex: string): HSV | null {
  if (!isHex(hex)) return null;
  let h = hex.trim().replace("#", "");
  if (h.length === 3) h = h.split("").map((c) => c + c).join("");
  const r = parseInt(h.slice(0, 2), 16) / 255;
  const g = parseInt(h.slice(2, 4), 16) / 255;
  const b = parseInt(h.slice(4, 6), 16) / 255;

  const max = Math.max(r, g, b);
  const min = Math.min(r, g, b);
  const d = max - min;

  let hue = 0;
  if (d !== 0) {
    if (max === r) hue = ((g - b) / d) % 6;
    else if (max === g) hue = (b - r) / d + 2;
    else hue = (r - g) / d + 4;
    hue *= 60;
    if (hue < 0) hue += 360;
  }
  return { h: hue, s: max === 0 ? 0 : d / max, v: max };
}

// lighten mixes a hex colour toward white by amount (0..1) to derive the "soft"
// accent variant for custom colours.
function lighten(hex: string, amount: number): string {
  let h = hex.trim().replace("#", "");
  if (h.length === 3) h = h.split("").map((c) => c + c).join("");
  const n = parseInt(h, 16);
  const r = (n >> 16) & 255;
  const g = (n >> 8) & 255;
  const b = n & 255;
  const mix = (c: number) => Math.round(c + (255 - c) * amount);
  const to2 = (c: number) => c.toString(16).padStart(2, "0");
  return `#${to2(mix(r))}${to2(mix(g))}${to2(mix(b))}`;
}

export function contrastText(hex:string):string {
 const value=hex.replace("#","");const h=value.length===3?value.split("").map(c=>c+c).join(""):value;
 const rgb=[0,2,4].map(i=>parseInt(h.slice(i,i+2),16)/255).map(c=>c<=.04045?c/12.92:Math.pow((c+.055)/1.055,2.4));
 const l=.2126*rgb[0]+.7152*rgb[1]+.0722*rgb[2];return (l+.05)/.05>1.05/(l+.05)?"#000000":"#ffffff";
}

// applyTheme writes the chosen appearance onto the document root. Called on
// startup and whenever an appearance setting changes.
export function applyTheme(opts: {
  theme: string;
  accent: string;
  font: string;
  radius: string;
  animation?: string;
  customThemes?: {id:string;name:string;colors:Record<string,string>;accent:string}[];
  uiScale?: number; density?: string; customRadius?: number;
}): void {
  const root = document.documentElement.style;

  const custom = opts.customThemes?.find(t => t.id === opts.theme);
  const preset = byId(THEME_PRESETS, opts.theme, THEME_PRESETS[0]);
  const c = {...preset.colors, ...(custom?.colors ?? {})};
  root.setProperty("--color-bg", c.bg);
  root.setProperty("--color-surface", c.surface);
  root.setProperty("--color-surface-2", c.surface2);
  root.setProperty("--color-border", c.border);
  root.setProperty("--color-border-soft", c.borderSoft);
  root.setProperty("--color-text", c.text);
  root.setProperty("--color-text-muted", c.textMuted);
  root.setProperty("--color-text-faint", c.textFaint);

  // Accent is either a palette id or a raw custom hex.
  if (isHex(opts.accent)) {
    root.setProperty("--color-accent", opts.accent);
    root.setProperty("--color-accent-soft", lighten(opts.accent, 0.3));
  } else {
    const accent = byId(ACCENTS, opts.accent, ACCENTS[0]);
    root.setProperty("--color-accent", accent.color);
    root.setProperty("--color-accent-soft", accent.soft);
  }

  root.setProperty("--color-ok", custom?.colors.success ?? "#5bd6a0");
  root.setProperty("--color-danger", custom?.colors.danger ?? "#ff6b6b");
  root.setProperty("font-size", `${16 * Math.min(120,Math.max(85,opts.uiScale ?? 100))/100}px`);
  document.documentElement.setAttribute("data-density",opts.density ?? "comfortable");
  root.setProperty("--color-on-accent", contrastText(isHex(opts.accent)?opts.accent:byId(ACCENTS,opts.accent,ACCENTS[0]).color));
  const font = byId(FONTS, opts.font, FONTS[0]);
  root.setProperty("--font-sans", font.stack);

  const radius = byId(RADII, opts.radius, RADII[1]);
  const rounding=opts.radius === "custom" ? Math.min(32,Math.max(0,opts.customRadius ?? 14)) : parseFloat(radius.value);
  for(const [name,multiplier] of Object.entries({xs:.25,sm:.35,md:.55,lg:1,xl:1.15,"2xl":1.3,"3xl":1.5}))root.setProperty(`--radius-${name}`,`${Math.round(rounding*multiplier*10)/10}px`);

  // Animations are selected by attribute; style.css holds one rule per preset.
  const anim = byId(ANIMATIONS, opts.animation ?? "", ANIMATIONS.find(a => a.id === "fade")!);
  document.documentElement.setAttribute("data-anim", anim.id);
}
