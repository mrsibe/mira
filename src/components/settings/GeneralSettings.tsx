import { Monitor, Moon, Sun } from "lucide-react";
import { useMemo } from "react";
import { useMessages } from "../../i18n";
import {
  type Appearance,
  buildFontOptions,
  chatSizeOptions,
  codeSizeOptions,
  defaultAppearance,
} from "../../lib/appearance";
import type { ThemeMode } from "../../lib/types";
import type { LanguageState } from "../../mygo";
import { SegmentedControl, Select } from "../../ui";

type GeneralSettingsProps = {
  language: LanguageState | null;
  onLanguageChange: (tag: string) => void;
  theme: ThemeMode;
  onThemeChange: (theme: ThemeMode) => void;
  appearance: Appearance;
  systemFonts: readonly string[];
  onAppearanceChange: (appearance: Appearance) => void;
};

export function GeneralSettings({
  language,
  onLanguageChange,
  theme,
  onThemeChange,
  appearance,
  systemFonts,
  onAppearanceChange,
}: GeneralSettingsProps) {
  const messages = useMessages();
  const interfaceFontOptions = useMemo(
    () => buildFontOptions("interface", systemFonts, appearance.interfaceFont, messages),
    [appearance.interfaceFont, messages, systemFonts],
  );
  const chatFontOptions = useMemo(
    () => buildFontOptions("chat", systemFonts, appearance.chatFont, messages),
    [appearance.chatFont, messages, systemFonts],
  );
  const codeFontOptions = useMemo(
    () => buildFontOptions("code", systemFonts, appearance.codeFont, messages),
    [appearance.codeFont, messages, systemFonts],
  );

  const themes = [
    { value: "light", label: messages.themeLight(), icon: Sun },
    { value: "dark", label: messages.themeDark(), icon: Moon },
    { value: "system", label: messages.themeSystem(), icon: Monitor },
  ] as const;
  const languageOptions = useMemo(
    () => [
      { value: "", label: messages.settingsLanguageSystem() },
      ...(language?.languages ?? []).map((item) => ({ value: item.tag, label: item.label })),
    ],
    [language, messages],
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
          title={messages.appearanceResetTitle()}
          onClick={resetAppearance}
        >
          {messages.appearanceReset()}
        </button>
      </div>

      <section className="settings-group">
        <h3 className="settings-group-title">{messages.settingsLanguage()}</h3>
        <p className="settings-description">{messages.settingsLanguageDescription()}</p>
        <div className="appearance-list">
          <div className="appearance-row">
            <span>{messages.settingsLanguage()}</span>
            <Select
              label={messages.settingsLanguage()}
              value={language?.chosen ?? ""}
              options={languageOptions}
              onChange={onLanguageChange}
            />
          </div>
        </div>
      </section>

      <section className="settings-group">
        <SegmentedControl
          label={messages.appearanceTheme()}
          value={theme}
          options={themes}
          optionClassName="theme-option"
          onChange={onThemeChange}
        />
      </section>

      <section className="settings-group">
        <h3 className="settings-group-title">{messages.appearanceInterface()}</h3>
        <div className="appearance-list">
          <div className="appearance-row">
            <span>{messages.appearanceInterfaceFont()}</span>
            <Select
              label={messages.appearanceInterfaceFont()}
              value={appearance.interfaceFont}
              options={interfaceFontOptions}
              onChange={(value) => patch("interfaceFont", value)}
            />
          </div>
        </div>
      </section>

      <section className="settings-group">
        <h3 className="settings-group-title">{messages.appearanceChat()}</h3>
        <div className="appearance-list">
          <div className="appearance-row">
            <span>{messages.appearanceChatFont()}</span>
            <Select
              label={messages.appearanceChatFont()}
              value={appearance.chatFont}
              options={chatFontOptions}
              onChange={(value) => patch("chatFont", value)}
            />
          </div>
          <div className="appearance-row">
            <span>{messages.appearanceChatSize()}</span>
            <Select
              label={messages.appearanceChatSize()}
              value={appearance.chatSize}
              options={chatSizeOptions}
              onChange={(value) => patch("chatSize", value)}
            />
          </div>
        </div>
      </section>

      <section className="settings-group">
        <h3 className="settings-group-title">{messages.appearanceCode()}</h3>
        <div className="appearance-list">
          <div className="appearance-row">
            <span>{messages.appearanceCodeFont()}</span>
            <Select
              label={messages.appearanceCodeFont()}
              value={appearance.codeFont}
              options={codeFontOptions}
              onChange={(value) => patch("codeFont", value)}
            />
          </div>
          <div className="appearance-row">
            <span>{messages.appearanceCodeSize()}</span>
            <Select
              label={messages.appearanceCodeSize()}
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
