import { Eye, EyeOff, KeyRound } from "lucide-react";
import { useMessages } from "../../i18n";
import type { View } from "../../mygo";
import { Button, Field, IconButton, Input, SegmentedControl, Switch } from "../../ui";
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
  const t = useMessages();
  return (
    <section className="settings-group">
      <h3 className="settings-group-title">{t.searchWebTitle()}</h3>
      <Switch
        checked={draft.webSearchEnabled}
        title={t.searchEnable()}
        description={t.searchEnableDescription()}
        onChange={() => setField("webSearchEnabled", !draft.webSearchEnabled)}
      />

      {draft.webSearchEnabled && (
        <>
          <div className="settings-subgroup">
            <span className="settings-label">{t.searchSource()}</span>
            <SegmentedControl
              label={t.searchProvider()}
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
                    <Input
                      id={id}
                      type={draft.showTavilyKey ? "text" : "password"}
                      value={draft.tavilyApiKey}
                      onChange={(event) => setField("tavilyApiKey", event.target.value)}
                      placeholder={
                        settings.hasTavilyApiKey ? t.searchTavilyKeySavedPlaceholder() : "tvly-…"
                      }
                      spellCheck={false}
                    />
                    <IconButton
                      label={draft.showTavilyKey ? t.commonHideKey() : t.commonShowKey()}
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
                      onError(error instanceof Error ? error.message : t.searchClearFailed());
                    }
                  }}
                >
                  {t.searchClearTavily()}
                </Button>
              )}
            </div>
          )}
        </>
      )}
    </section>
  );
}
