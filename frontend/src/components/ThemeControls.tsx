import { useState } from "react";
import { modes, setMode, setTheme, storedMode, storedTheme, themes } from "../theme";
import type { Mode, Theme } from "../theme";

function label(value: string) {
  return value.charAt(0).toUpperCase() + value.slice(1);
}

export default function ThemeControls() {
  const [theme, updateTheme] = useState<Theme>(storedTheme);
  const [mode, updateMode] = useState<Mode>(storedMode);

  return (
    <div className="flex justify-end gap-2 mb-2">
      <select
        aria-label="Theme"
        value={theme}
        onChange={(e) => {
          const next = e.target.value as Theme;
          setTheme(next);
          updateTheme(next);
        }}
        className="ap-input px-2 py-1 text-sm border border-gray-300 rounded-lg"
      >
        {themes.map((t) => (
          <option key={t} value={t}>
            {label(t)}
          </option>
        ))}
      </select>
      <select
        aria-label="Light or dark"
        value={mode}
        disabled={theme === "oldschool"}
        onChange={(e) => {
          const next = e.target.value as Mode;
          setMode(next);
          updateMode(next);
        }}
        className="ap-input px-2 py-1 text-sm border border-gray-300 rounded-lg disabled:opacity-50"
      >
        {modes.map((m) => (
          <option key={m} value={m}>
            {label(m)}
          </option>
        ))}
      </select>
    </div>
  );
}
