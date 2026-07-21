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

// Six dark presets, no harsh pure white text. "graphite" is the shipped default.
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
];

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

function byId<T extends { id: string }>(list: T[], id: string, fallback: T): T {
  return list.find((x) => x.id === id) ?? fallback;
}

// isHex reports whether a string is a #rgb / #rrggbb colour.
export function isHex(v: string): boolean {
  return /^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/.test(v.trim());
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

// applyTheme writes the chosen appearance onto the document root. Called on
// startup and whenever an appearance setting changes.
export function applyTheme(opts: {
  theme: string;
  accent: string;
  font: string;
  radius: string;
}): void {
  const root = document.documentElement.style;

  const preset = byId(THEME_PRESETS, opts.theme, THEME_PRESETS[0]);
  const c = preset.colors;
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

  const font = byId(FONTS, opts.font, FONTS[0]);
  root.setProperty("--font-sans", font.stack);

  const radius = byId(RADII, opts.radius, RADII[1]);
  root.setProperty("--radius-lg", radius.value);
}
