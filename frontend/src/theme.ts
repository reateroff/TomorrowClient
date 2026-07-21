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

// Three calm dark greys, no pure black. "graphite" is the shipped default.
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
];

export interface Accent {
  id: string;
  name: string;
  color: string;
  soft: string;
}

// A restrained six-colour accent palette; the default indigo matches 1.0.0.
export const ACCENTS: Accent[] = [
  { id: "indigo", name: "Индиго", color: "#7c8cff", soft: "#a5b0ff" },
  { id: "teal", name: "Бирюза", color: "#4fd1c5", soft: "#7ee0d7" },
  { id: "emerald", name: "Изумруд", color: "#5bd6a0", soft: "#84e4bb" },
  { id: "amber", name: "Янтарь", color: "#e0b155", soft: "#ecc987" },
  { id: "rose", name: "Роза", color: "#f08a9c", soft: "#f5adba" },
  { id: "violet", name: "Фиалка", color: "#b18cff", soft: "#c9b0ff" },
];

export interface FontChoice {
  id: string;
  name: string;
  stack: string;
}

// Font stacks; the bundled families load via @font-face in style.css.
export const FONTS: FontChoice[] = [
  { id: "inter", name: "Inter", stack: '"Inter", "Segoe UI", system-ui, sans-serif' },
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

  const accent = byId(ACCENTS, opts.accent, ACCENTS[0]);
  root.setProperty("--color-accent", accent.color);
  root.setProperty("--color-accent-soft", accent.soft);

  const font = byId(FONTS, opts.font, FONTS[0]);
  root.setProperty("--font-sans", font.stack);

  const radius = byId(RADII, opts.radius, RADII[1]);
  root.setProperty("--radius-lg", radius.value);
}
