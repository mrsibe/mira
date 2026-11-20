import { type FormEvent, useCallback, useEffect, useState } from "react";
import { useMessages } from "../i18n";
import type { Appearance } from "../lib/appearance";
import type { SettingsSection, ThemeMode } from "../lib/types";
import { type Input, type LanguageState, Mira, type ModelView, type View } from "../mygo";
import { Button, Modal, Notice } from "../ui";
import { AboutSettings } from "./settings/AboutSettings";
import { GeneralSettings } from "./settings/GeneralSettings";
import { MemorySettings } from "./settings/MemorySettings";
import { ModelSettings } from "./settings/ModelSettings";
import { PersonalizationSettings } from "./settings/PersonalizationSettings";
import { SearchSettings } from "./settings/SearchSettings";
import { SettingsNav } from "./settings/SettingsNav";
import type { SettingsDraft } from "./settings/types";
import { useMemoryEditor } from "./settings/useMemoryEditor";

type SettingsDialogProps = {
  settings: View;
  language: LanguageState | null;
  theme: ThemeMode;
  appearance: Appearance;
  initialSection: SettingsSection;
  onClose: () => void;
  onSave: (input: Input) => Promise<View>;
  onActivateModel: (id: string) => Promise<View>;
  onAddModel: () => void;
  onEditModel: (model: ModelView) => void;
  onDeleteModel: (id: string) => Promise<View>;
  onClearTavilyKey: () => Promise<View>;
  onThemeChange: (theme: ThemeMode) => void;
  onAppearanceChange: (appearance: Appearance) => void;
  onLanguageChange: (tag: string) => void;
};

export function SettingsDialog({
  settings,
  language,
  theme,
  appearance,
  initialSection,
  onClose,
  onSave,
  onActivateModel,
  onAddModel,
  onEditModel,
  onDeleteModel,
  onClearTavilyKey,
  onThemeChange,
  onAppearanceChange,
  onLanguageChange,
}: SettingsDialogProps) {
  const t = useMessages();
  const [section, setSection] = useState<SettingsSection>(initialSection);
  const [draft, setDraft] = useState<SettingsDraft>(() => createDraft(settings));
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [saving, setSaving] = useState(false);
  const [systemFonts, setSystemFonts] = useState<string[]>([]);
  const memory = useMemoryEditor({
    active: section === "memory",
    onError: setError,
  });

  useEffect(() => {
    let active = true;
    void Mira.listFonts()
      .then((fonts) => {
        if (active) setSystemFonts(fonts);
      })
      .catch(() => {
        // Preset fonts remain available when system font discovery fails.
      });
    return () => {
      active = false;
    };
  }, []);

  const setField = useCallback(<K extends keyof SettingsDraft>(key: K, value: SettingsDraft[K]) => {
    setDraft((current) => ({ ...current, [key]: value }));
  }, []);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setError("");
    setNotice("");
    if (
      draft.webSearchEnabled &&
      draft.searchProvider === "tavily" &&
      !settings.hasTavilyApiKey &&
      !draft.tavilyApiKey.trim()
    ) {
      setError(t.settingsTavilyKeyRequired());
      setSection("search");
      return;
    }

    const enteredTavilyKey = draft.tavilyApiKey.trim() !== "";
    setSaving(true);
    try {
      const saved = await onSave({
        userName: draft.userName,
        webSearchEnabled: draft.webSearchEnabled,
        searchProvider: draft.searchProvider,
        tavilyApiKey: draft.tavilyApiKey,
        responseStyle: draft.responseStyle,
        personality: draft.personality,
        warmth: draft.warmth,
        enthusiasm: draft.enthusiasm,
        emoji: draft.emoji,
        customInstructions: draft.customInstructions,
        memoryEnabled: draft.memoryEnabled,
        memorySummaryEnabled: draft.memorySummaryEnabled,
        memorySearchEnabled: draft.memorySearchEnabled,
        intelligentUIEnabled: draft.intelligentUIEnabled,
      });
      setField("tavilyApiKey", "");
      if (memory.dirty) {
        try {
          await memory.saveDraft();
        } catch (saveError) {
          setError(saveError instanceof Error ? saveError.message : t.settingsSaveMemoryFailed());
          setSection("memory");
          return;
        }
      }
      if (enteredTavilyKey && saved.tavilyApiKeyStorage === "session") {
        setNotice(t.settingsKeychainUnavailableNotice());
        return;
      }
      onClose();
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : t.settingsSaveFailed());
    } finally {
      setSaving(false);
    }
  }

  return (
    <Modal className="settings-modal" labelledBy="settings-title" onClose={onClose}>
      <div className="settings-layout">
        <SettingsNav section={section} onChange={setSection} />
        <form className="settings-panel" onSubmit={submit}>
          <h2 id="settings-title" className="sr-only">
            {t.settingsTitle()}
          </h2>
          <div className="settings-content">
            {section === "general" && (
              <GeneralSettings
                theme={theme}
                onThemeChange={onThemeChange}
                appearance={appearance}
                systemFonts={systemFonts}
                onAppearanceChange={onAppearanceChange}
                language={language}
                onLanguageChange={onLanguageChange}
              />
            )}
            {section === "personalization" && (
              <PersonalizationSettings draft={draft} setField={setField} />
            )}
            {section === "memory" && (
              <MemorySettings draft={draft} setField={setField} editor={memory} />
            )}
            {section === "search" && (
              <SearchSettings
                settings={settings}
                draft={draft}
                setField={setField}
                onClearTavilyKey={onClearTavilyKey}
                onError={setError}
              />
            )}
            {section === "model" && (
              <ModelSettings
                settings={settings}
                onActivateModel={onActivateModel}
                onAddModel={onAddModel}
                onEditModel={onEditModel}
                onDeleteModel={onDeleteModel}
                onError={setError}
              />
            )}
            {section === "about" && <AboutSettings />}

            {error && <Notice tone="error">{error}</Notice>}
            {notice && <Notice>{notice}</Notice>}
          </div>

          <footer className="modal-actions settings-footer">
            <span className="spacer" />
            {section === "about" || section === "general" || section === "model" ? (
              <Button onClick={onClose}>{t.commonDone()}</Button>
            ) : (
              <>
                <Button onClick={onClose}>{t.commonCancel()}</Button>
                <Button type="submit" variant="primary" disabled={saving || memory.action}>
                  {saving
                    ? t.commonSaving()
                    : memory.dirty && section === "memory"
                      ? t.settingsSaveMemory()
                      : t.commonSave()}
                </Button>
              </>
            )}
          </footer>
        </form>
      </div>
    </Modal>
  );
}

function createDraft(settings: View): SettingsDraft {
  return {
    userName: settings.userName,
    webSearchEnabled: settings.webSearchEnabled,
    searchProvider: settings.searchProvider === "tavily" ? "tavily" : "duckduckgo",
    tavilyApiKey: "",
    showTavilyKey: false,
    responseStyle: settings.responseStyle,
    personality: settings.personality,
    warmth: settings.warmth,
    enthusiasm: settings.enthusiasm,
    emoji: settings.emoji,
    customInstructions: settings.customInstructions,
    memoryEnabled: settings.memoryEnabled,
    memorySummaryEnabled: settings.memorySummaryEnabled,
    memorySearchEnabled: settings.memorySearchEnabled,
    intelligentUIEnabled: settings.intelligentUIEnabled,
  };
}
