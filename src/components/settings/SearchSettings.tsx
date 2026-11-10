import { Eye, EyeOff, KeyRound } from "lucide-react";
import type { View } from "../../mygo";
import { Button, Field, IconButton, SegmentedControl, Switch } from "../../ui";
import { KeyStorageStatus } from "./KeyStorageStatus";
import type { SetDraftField, SettingsDraft } from "./types";

const providers = [
  { value: "duckduckgo", label: "DuckDuckGo" },
  { value: "tavily", label: "Tavily" },
] as const;

type SearchSettingsProps = {
  settings: View;
  draft: SettingsDraft;
  setField: SetDraftField;
  onClearTavilyKey: () => Promise<View>;
  onError: (message: string) => void;
};

export function SearchSettings({
  settings,
  draft,
  setField,
  onClearTavilyKey,
  onError,
}: SearchSettingsProps) {
  return (
    <section className="settings-group">
      <h3 className="settings-group-title">联网搜索</h3>
      <Switch
        checked={draft.webSearchEnabled}
        title="启用"
        description="需要最新信息时搜索网页并引用来源。"
        onChange={() => setField("webSearchEnabled", !draft.webSearchEnabled)}
      />

      {draft.webSearchEnabled && (
        <>
          <div className="settings-subgroup">
            <span className="settings-label">搜索来源</span>
            <SegmentedControl
              label="搜索供应商"
              value={draft.searchProvider}
              options={providers}
              className="search-provider-options"
              optionClassName="theme-option"
              onChange={(value) => setField("searchProvider", value)}
            />
          </div>

          {draft.searchProvider === "tavily" && (
            <div className="settings-subgroup">
              <Field label="Tavily API Key">
                {(id) => (
                  <div className="key-input">
                    <KeyRound size={15} />
                    <input
                      id={id}
                      type={draft.showTavilyKey ? "text" : "password"}
                      value={draft.tavilyApiKey}
                      onChange={(event) => setField("tavilyApiKey", event.target.value)}
                      placeholder={settings.hasTavilyApiKey ? "已保存，留空表示不修改" : "tvly-…"}
                      spellCheck={false}
                    />
                    <IconButton
                      label={draft.showTavilyKey ? "隐藏密钥" : "显示密钥"}
                      onClick={() => setField("showTavilyKey", !draft.showTavilyKey)}
                    >
                      {draft.showTavilyKey ? <EyeOff size={15} /> : <Eye size={15} />}
                    </IconButton>
                  </div>
                )}
              </Field>
              <KeyStorageStatus
                storage={settings.tavilyApiKeyStorage}
                keychainAvailable={settings.keychainAvailable}
                keychainError={settings.keychainError}
              />
              {settings.hasTavilyApiKey && settings.tavilyApiKeyStorage !== "environment" && (
                <Button
                  variant="dangerText"
                  className="settings-clear-key"
                  onClick={async () => {
                    onError("");
                    try {
                      const saved = await onClearTavilyKey();
                      setField(
                        "searchProvider",
                        saved.searchProvider === "tavily" ? "tavily" : "duckduckgo",
                      );
                      setField("tavilyApiKey", "");
                    } catch (error) {
                      onError(error instanceof Error ? error.message : "清除失败");
                    }
                  }}
                >
                  清除已保存的 Tavily Key
                </Button>
              )}
            </div>
          )}
        </>
      )}
    </section>
  );
}
