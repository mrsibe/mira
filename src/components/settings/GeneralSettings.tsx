import { Monitor, Moon, Sun } from "lucide-react";
import type { ThemeMode } from "../../lib/types";
import { SegmentedControl } from "../../ui";

const themes = [
  { value: "light", label: "浅色", icon: Sun },
  { value: "dark", label: "深色", icon: Moon },
  { value: "system", label: "跟随系统", icon: Monitor },
] as const;

type GeneralSettingsProps = {
  theme: ThemeMode;
  onThemeChange: (theme: ThemeMode) => void;
};

export function GeneralSettings({ theme, onThemeChange }: GeneralSettingsProps) {
  return (
    <section className="settings-group">
      <SegmentedControl
        label="主题"
        value={theme}
        options={themes}
        optionClassName="theme-option"
        onChange={onThemeChange}
      />
    </section>
  );
}
