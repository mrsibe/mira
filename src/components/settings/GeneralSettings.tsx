import { Monitor, Moon, Sun } from "lucide-react";
import type { ThemeMode } from "../../lib/types";
import { Field, SegmentedControl } from "../../ui";
import type { SetDraftField, SettingsDraft } from "./types";

const themes = [
  { value: "light", label: "浅色", icon: Sun },
  { value: "dark", label: "深色", icon: Moon },
  { value: "system", label: "跟随系统", icon: Monitor },
] as const;

type GeneralSettingsProps = {
  draft: SettingsDraft;
  setField: SetDraftField;
  theme: ThemeMode;
  onThemeChange: (theme: ThemeMode) => void;
};

export function GeneralSettings({ draft, setField, theme, onThemeChange }: GeneralSettingsProps) {
  return (
    <>
      <section className="settings-group">
        <h3 className="settings-group-title">常规</h3>
        <Field label="称呼">
          {(id) => (
            <input
              id={id}
              value={draft.userName}
              maxLength={32}
              onChange={(event) => setField("userName", event.target.value)}
              placeholder="留空则不显示"
            />
          )}
        </Field>
      </section>

      <section className="settings-group">
        <h3 className="settings-group-title">主题</h3>
        <SegmentedControl
          label="主题"
          value={theme}
          options={themes}
          optionClassName="theme-option"
          onChange={onThemeChange}
        />
      </section>
    </>
  );
}
