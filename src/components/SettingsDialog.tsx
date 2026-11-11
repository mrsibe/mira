import { type FormEvent, useCallback, useState } from "react";
import type { SettingsSection, ThemeMode } from "../lib/types";
import type { Input, View } from "../mygo";
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
  theme: ThemeMode;
  initialSection: SettingsSection;
  onClose: () => void;
  onSave: (input: Input) => Promise<View>;
  onClearKey: () => Promise<View>;
  onClearTavilyKey: () => Promise<View>;
  onThemeChange: (theme: ThemeMode) => void;
};

export function SettingsDialog({
  settings,
  theme,
  initialSection,
  onClose,
  onSave,
  onClearKey,
  onClearTavilyKey,
  onThemeChange,
}: SettingsDialogProps) {
  const [section, setSection] = useState<SettingsSection>(initialSection);
  const [draft, setDraft] = useState<SettingsDraft>(() => createDraft(settings));
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [saving, setSaving] = useState(false);
  const memory = useMemoryEditor({
    active: section === "memory",
    onError: setError,
  });

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
      setError("使用 Tavily 搜索时需要填写 API Key");
      setSection("search");
      return;
    }

    const enteredModelKey = draft.apiKey.trim() !== "";
    const enteredTavilyKey = draft.tavilyApiKey.trim() !== "";
    setSaving(true);
    try {
      const saved = await onSave({
        baseUrl: draft.baseUrl,
        model: draft.model,
        protocol: draft.protocol,
        apiKey: draft.apiKey,
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
      setField("apiKey", "");
      setField("tavilyApiKey", "");
      if (memory.dirty) {
        try {
          await memory.saveDraft();
        } catch (saveError) {
          setError(saveError instanceof Error ? saveError.message : "保存记忆失败");
          setSection("memory");
          return;
        }
      }
      if (
        (enteredModelKey && saved.apiKeyStorage === "session") ||
        (enteredTavilyKey && saved.tavilyApiKeyStorage === "session")
      ) {
        setNotice("系统钥匙串不可用，刚输入的密钥只保存在本次会话，重启后需要重新输入。");
        return;
      }
      onClose();
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : "保存失败");
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
            设置
          </h2>
          <div className="settings-content">
            {section === "general" && (
              <GeneralSettings theme={theme} onThemeChange={onThemeChange} />
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
                draft={draft}
                setField={setField}
                onClearKey={onClearKey}
                onError={setError}
              />
            )}
            {section === "about" && <AboutSettings />}

            {error && <Notice tone="error">{error}</Notice>}
            {notice && <Notice>{notice}</Notice>}
          </div>

          <footer className="modal-actions settings-footer">
            <span className="spacer" />
            {section === "about" ? (
              <Button onClick={onClose}>关闭</Button>
            ) : (
              <>
                <Button onClick={onClose}>取消</Button>
                <Button type="submit" variant="primary" disabled={saving || memory.action}>
                  {saving ? "保存中…" : memory.dirty && section === "memory" ? "保存记忆" : "保存"}
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
    baseUrl: settings.baseUrl,
    model: settings.model,
    protocol: settings.protocol,
    userName: settings.userName,
    apiKey: "",
    showKey: false,
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
