import { Eye, EyeOff, KeyRound } from "lucide-react";
import type { View } from "../../mygo";
import { Button, Field, IconButton } from "../../ui";
import { KeyStorageStatus } from "./KeyStorageStatus";
import type { SetDraftField, SettingsDraft } from "./types";

type ModelSettingsProps = {
  settings: View;
  draft: SettingsDraft;
  setField: SetDraftField;
  onClearKey: () => Promise<View>;
  onError: (message: string) => void;
};

export function ModelSettings({
  settings,
  draft,
  setField,
  onClearKey,
  onError,
}: ModelSettingsProps) {
  return (
    <section className="settings-group">
      <h3 className="settings-group-title">API 连接</h3>
      <p className="settings-description">兼容 OpenAI API。</p>
      <Field label="Base URL">
        {(id) => (
          <input
            id={id}
            value={draft.baseUrl}
            onChange={(event) => setField("baseUrl", event.target.value)}
            placeholder="https://api.openai.com/v1"
            spellCheck={false}
          />
        )}
      </Field>
      <div className="field-row">
        <Field label="模型">
          {(id) => (
            <input
              id={id}
              value={draft.model}
              onChange={(event) => setField("model", event.target.value)}
              placeholder="gpt-5"
              spellCheck={false}
            />
          )}
        </Field>
        <Field label="API 协议">
          {(id) => (
            <select
              id={id}
              value={draft.protocol}
              onChange={(event) => setField("protocol", event.target.value)}
            >
              <option value="responses">Responses</option>
              <option value="completions">Chat Completions</option>
            </select>
          )}
        </Field>
      </div>
      <Field label="API Key">
        {(id) => (
          <div className="key-input">
            <KeyRound size={15} />
            <input
              id={id}
              type={draft.showKey ? "text" : "password"}
              value={draft.apiKey}
              onChange={(event) => setField("apiKey", event.target.value)}
              placeholder={
                settings.apiKeyStorage === "environment"
                  ? "来自环境变量；输入新密钥可覆盖"
                  : settings.hasApiKey
                    ? "已配置，留空表示不修改"
                    : "sk-…"
              }
              spellCheck={false}
            />
            <IconButton
              label={draft.showKey ? "隐藏密钥" : "显示密钥"}
              onClick={() => setField("showKey", !draft.showKey)}
            >
              {draft.showKey ? <EyeOff size={15} /> : <Eye size={15} />}
            </IconButton>
          </div>
        )}
      </Field>
      <KeyStorageStatus
        storage={settings.apiKeyStorage}
        keychainAvailable={settings.keychainAvailable}
        keychainError={settings.keychainError}
      />
      {settings.hasApiKey && settings.apiKeyStorage !== "environment" && (
        <Button
          variant="dangerText"
          className="settings-clear-key"
          onClick={async () => {
            onError("");
            try {
              await onClearKey();
            } catch (error) {
              onError(error instanceof Error ? error.message : "清除失败");
            }
          }}
        >
          清除已保存的密钥
        </Button>
      )}
    </section>
  );
}
