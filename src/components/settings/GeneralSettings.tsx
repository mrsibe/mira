import { Monitor, Moon, Sun } from "lucide-react";
import { useMemo } from "react";
import {
  type Appearance,
  buildFontOptions,
  chatSizeOptions,
  codeSizeOptions,
  defaultAppearance,
} from "../../lib/appearance";
import type { ThemeMode } from "../../lib/types";
import { SegmentedControl, Select } from "../../ui";

const themes = [
  { value: "light", label: "浅色", icon: Sun },
  { value: "dark", label: "深色", icon: Moon },
  { value: "system", label: "跟随系统", icon: Monitor },
] as const;

type GeneralSettingsProps = {
  theme: ThemeMode;
  onThemeChange: (theme: ThemeMode) => void;
  appearance: Appearance;
  systemFonts: readonly string[];
  onAppearanceChange: (appearance: Appearance) => void;
};

export function GeneralSettings({
  theme,
  onThemeChange,
  appearance,
  systemFonts,
  onAppearanceChange,
}: GeneralSettingsProps) {
  const interfaceFontOptions = useMemo(
    () => buildFontOptions("interface", systemFonts, appearance.interfaceFont),
    [appearance.interfaceFont, systemFonts],
  );
  const chatFontOptions = useMemo(
    () => buildFontOptions("chat", systemFonts, appearance.chatFont),
    [appearance.chatFont, systemFonts],
  );
  const codeFontOptions = useMemo(
    () => buildFontOptions("code", systemFonts, appearance.codeFont),
    [appearance.codeFont, systemFonts],
  );

  function patch<K extends keyof Appearance>(key: K, value: Appearance[K]) {
    onAppearanceChange({ ...appearance, [key]: value });
  }

  function resetAppearance() {
    onAppearanceChange(defaultAppearance);
    onThemeChange("system");
  }

  return (
    <>
      <div className="appearance-toolbar">
        <button
          type="button"
          className="appearance-reset"
          title="重置主题、字体和字号；不会影响模型、搜索、记忆和个性化设置"
          onClick={resetAppearance}
        >
          恢复默认
        </button>
      </div>

      <section className="settings-group">
        <SegmentedControl
          label="主题"
          value={theme}
          options={themes}
          optionClassName="theme-option"
          onChange={onThemeChange}
        />
      </section>

      <section className="settings-group">
        <h3 className="settings-group-title">界面</h3>
        <div className="appearance-list">
          <div className="appearance-row">
            <span>界面字体</span>
            <Select
              label="界面字体"
              value={appearance.interfaceFont}
              options={interfaceFontOptions}
              onChange={(value) => patch("interfaceFont", value)}
            />
          </div>
        </div>
      </section>

      <section className="settings-group">
        <h3 className="settings-group-title">对话</h3>
        <div className="appearance-list">
          <div className="appearance-row">
            <span>对话字体</span>
            <Select
              label="对话字体"
              value={appearance.chatFont}
              options={chatFontOptions}
              onChange={(value) => patch("chatFont", value)}
            />
          </div>
          <div className="appearance-row">
            <span>对话字号</span>
            <Select
              label="对话字号"
              value={appearance.chatSize}
              options={chatSizeOptions}
              onChange={(value) => patch("chatSize", value)}
            />
          </div>
        </div>
      </section>

      <section className="settings-group">
        <h3 className="settings-group-title">代码</h3>
        <div className="appearance-list">
          <div className="appearance-row">
            <span>代码字体</span>
            <Select
              label="代码字体"
              value={appearance.codeFont}
              options={codeFontOptions}
              onChange={(value) => patch("codeFont", value)}
            />
          </div>
          <div className="appearance-row">
            <span>代码字号</span>
            <Select
              label="代码字号"
              value={appearance.codeSize}
              options={codeSizeOptions}
              onChange={(value) => patch("codeSize", value)}
            />
          </div>
        </div>
      </section>
    </>
  );
}
