export const themes = ["oldschool", "classic", "modern"] as const;
export const modes = ["auto", "light", "dark"] as const;

export type Theme = (typeof themes)[number];
export type Mode = (typeof modes)[number];

const THEME_KEY = "theme";
const MODE_KEY = "mode";

export function storedTheme(): Theme {
  const value = localStorage.getItem(THEME_KEY);
  return themes.find((t) => t === value) ?? "oldschool";
}

export function storedMode(): Mode {
  const value = localStorage.getItem(MODE_KEY);
  return modes.find((m) => m === value) ?? "auto";
}

function systemDark(): boolean {
  return window.matchMedia?.("(prefers-color-scheme: dark)").matches ?? false;
}

// Oldschool only exists in light; the other themes follow the stored mode.
export function applyTheme() {
  const root = document.documentElement;
  const theme = storedTheme();
  const mode = storedMode();
  root.dataset.theme = theme;
  root.dataset.mode = theme === "oldschool" ? "light" : mode === "auto" ? (systemDark() ? "dark" : "light") : mode;
}

export function setTheme(theme: Theme) {
  localStorage.setItem(THEME_KEY, theme);
  applyTheme();
}

export function setMode(mode: Mode) {
  localStorage.setItem(MODE_KEY, mode);
  applyTheme();
}

// Auto has to follow a change of the system setting while the page is open.
export function watchSystemMode() {
  window.matchMedia?.("(prefers-color-scheme: dark)").addEventListener("change", applyTheme);
}
