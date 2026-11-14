import { ArrowLeft, Eye, EyeOff, KeyRound } from "lucide-react";
import { useId, useState } from "react";
import type { ModelInput, ModelView, ProviderInfo, View } from "../mygo";
import { Button, Field, IconButton, Input, Modal, Notice, Select } from "../ui";
import { KeyStorageStatus } from "./settings/KeyStorageStatus";

type ModelWizardProps = {
  mode: "add" | "edit";
  providers: ProviderInfo[];
  initialModel?: ModelView;
  firstRun?: boolean;
  keychainAvailable: boolean;
  keychainError?: string;
  onClose: () => void;
  onSkip?: () => void;
  onTest: (input: ModelInput) => Promise<void>;
  onSubmit: (input: ModelInput) => Promise<View>;
  onClearKey?: (id: string) => Promise<View>;
};

type Draft = {
  name: string;
  provider: string;
  baseUrl: string;
  model: string;
  protocol: string;
  apiKey: string;
};

export function ModelWizard({
  mode,
  providers,
  initialModel,
  firstRun = false,
  keychainAvailable,
  keychainError,
  onClose,
  onSkip,
  onTest,
  onSubmit,
  onClearKey,
}: ModelWizardProps) {
  const titleId = useId();
  const [step, setStep] = useState<"provider" | "config">(mode === "edit" ? "config" : "provider");
  const [draft, setDraft] = useState<Draft>(() => initialDraft(providers, initialModel));
  const [showKey, setShowKey] = useState(false);
  const [hasKey, setHasKey] = useState(initialModel?.hasApiKey ?? false);
  const [keyStorage, setKeyStorage] = useState(initialModel?.apiKeyStorage ?? "none");
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [testing, setTesting] = useState(false);
  const [saving, setSaving] = useState(false);
  const [clearing, setClearing] = useState(false);

  const provider = providers.find((item) => item.id === draft.provider);
  const busy = testing || saving || clearing;

  function setField<K extends keyof Draft>(key: K, value: Draft[K]) {
    setDraft((current) => ({ ...current, [key]: value }));
    setError("");
    setNotice("");
  }

  function chooseProvider(next: ProviderInfo) {
    setDraft((current) => ({
      ...current,
      provider: next.id,
      baseUrl: next.defaultBaseUrl,
      model: next.defaultModel,
      protocol: next.defaultProtocol,
      apiKey: "",
    }));
    setHasKey(false);
    setKeyStorage("none");
    setError("");
    setNotice("");
  }

  function input(): ModelInput {
    return {
      id: initialModel?.id,
      name: draft.name.trim(),
      provider: draft.provider,
      baseUrl: draft.baseUrl.trim(),
      model: draft.model.trim(),
      protocol: draft.protocol,
      apiKey: draft.apiKey.trim(),
    };
  }

  function validationError() {
    if (!provider) return "请选择服务商";
    if (!draft.model.trim()) return `${provider.modelLabel || "模型"}不能为空`;
    if (provider.requiresBaseUrl && !draft.baseUrl.trim()) return "Base URL 不能为空";
    const local = isLocalURL(draft.baseUrl);
    const hasUsableKey = draft.apiKey.trim() !== "" || hasKey || keyStorage === "environment";
    if (provider.requiresApiKey && !(provider.localAuthOptional && local) && !hasUsableKey) {
      return "请填写 API Key";
    }
    return "";
  }

  async function testConnection() {
    const message = validationError();
    if (message) {
      setError(message);
      return;
    }
    setTesting(true);
    setError("");
    setNotice("");
    try {
      await onTest(input());
      setNotice("连接成功，模型响应正常。");
    } catch (testError) {
      setError(testError instanceof Error ? testError.message : "连接失败");
    } finally {
      setTesting(false);
    }
  }

  async function submit() {
    const message = validationError();
    if (message) {
      setError(message);
      return;
    }
    setSaving(true);
    setError("");
    setNotice("");
    try {
      await onSubmit(input());
      setSaving(false);
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : "保存模型失败");
      setSaving(false);
    }
  }

  async function clearKey() {
    if (!initialModel || !onClearKey) return;
    setClearing(true);
    setError("");
    setNotice("");
    try {
      await onClearKey(initialModel.id);
      setHasKey(false);
      setKeyStorage("none");
      setNotice("已清除该模型的 API Key。");
    } catch (clearError) {
      setError(clearError instanceof Error ? clearError.message : "清除密钥失败");
    } finally {
      setClearing(false);
    }
  }

  const title = mode === "edit" ? "编辑模型" : firstRun ? "添加第一个模型" : "添加模型";

  return (
    <Modal
      className="model-wizard-modal"
      labelledBy={titleId}
      onClose={() => {
        if (!busy) onClose();
      }}
    >
      <header className="modal-head model-wizard-head">
        <div>
          <h2 id={titleId}>{title}</h2>
          <p>
            {step === "provider"
              ? "选择模型服务商，下一步填写连接信息。"
              : mode === "edit"
                ? "修改连接信息后保存。服务商保持不变。"
                : "连接信息会立即生效，并自动设为当前模型。"}
          </p>
        </div>
        <span className="model-wizard-step">{step === "provider" ? "1 / 2" : "2 / 2"}</span>
      </header>

      <div className="model-wizard-body">
        {step === "provider" ? (
          <div className="model-wizard-provider-select">
            <Field label="服务商">
              {(id) => (
                <Select
                  id={id}
                  className="settings-select"
                  label="服务商"
                  value={draft.provider}
                  options={providers.map((item) => ({
                    value: item.id,
                    label: item.label,
                  }))}
                  onChange={(value) => {
                    const next = providers.find((item) => item.id === value);
                    if (next) chooseProvider(next);
                  }}
                />
              )}
            </Field>
            {provider && <p className="settings-description">{provider.description}</p>}
          </div>
        ) : (
          <div className="model-wizard-form">
            <div className="model-wizard-provider">
              <div>
                <span className="settings-label">服务商</span>
                <strong>{provider?.label ?? draft.provider}</strong>
              </div>
              {mode === "add" && (
                <Button disabled={busy} onClick={() => setStep("provider")}>
                  <ArrowLeft size={15} />
                  更换
                </Button>
              )}
            </div>

            <Field label="名称（可选）">
              {(id) => (
                <Input
                  id={id}
                  value={draft.name}
                  placeholder={draft.model || "例如：主力模型"}
                  onChange={(event) => setField("name", event.target.value)}
                />
              )}
            </Field>

            <Field label={provider?.id === "azure" ? "Endpoint" : "Base URL"}>
              {(id) => (
                <Input
                  id={id}
                  value={draft.baseUrl}
                  placeholder={provider?.baseUrlHint || "https://example.com/v1"}
                  spellCheck={false}
                  onChange={(event) => setField("baseUrl", event.target.value)}
                />
              )}
            </Field>
            {provider?.baseUrlHint && <p className="model-wizard-hint">{provider.baseUrlHint}</p>}

            <div className="field-row">
              <Field label={provider?.modelLabel || "模型"}>
                {(id) => (
                  <Input
                    id={id}
                    value={draft.model}
                    placeholder={provider?.defaultModel || "模型名称"}
                    spellCheck={false}
                    onChange={(event) => setField("model", event.target.value)}
                  />
                )}
              </Field>
              {provider && provider.protocols.length > 1 && (
                <Field label="API 协议">
                  {(id) => (
                    <Select
                      id={id}
                      className="settings-select"
                      label="API 协议"
                      value={draft.protocol}
                      options={provider.protocols}
                      onChange={(value) => setField("protocol", value)}
                    />
                  )}
                </Field>
              )}
            </div>

            <Field label="API Key">
              {(id) => (
                <div className="key-input">
                  <KeyRound size={15} />
                  <Input
                    id={id}
                    type={showKey ? "text" : "password"}
                    value={draft.apiKey}
                    placeholder={
                      keyStorage === "environment"
                        ? "来自环境变量；输入新密钥可覆盖"
                        : hasKey
                          ? "已配置，留空表示不修改"
                          : provider?.localAuthOptional
                            ? "本地无鉴权服务可留空"
                            : "sk-…"
                    }
                    spellCheck={false}
                    onChange={(event) => setField("apiKey", event.target.value)}
                  />
                  <IconButton
                    label={showKey ? "隐藏密钥" : "显示密钥"}
                    onClick={() => setShowKey((visible) => !visible)}
                  >
                    {showKey ? <EyeOff size={15} /> : <Eye size={15} />}
                  </IconButton>
                </div>
              )}
            </Field>
            <KeyStorageStatus
              storage={keyStorage}
              keychainAvailable={keychainAvailable}
              keychainError={keychainError}
            />
            {hasKey && keyStorage !== "environment" && onClearKey && (
              <Button
                variant="dangerText"
                className="settings-clear-key"
                disabled={busy}
                onClick={() => void clearKey()}
              >
                清除已保存的密钥
              </Button>
            )}

            {error && <Notice tone="error">{error}</Notice>}
            {notice && <Notice>{notice}</Notice>}
          </div>
        )}
      </div>

      <footer className="modal-actions model-wizard-actions">
        <span className="spacer" />
        {step === "provider" ? (
          <>
            <Button disabled={busy} onClick={onSkip ?? onClose}>
              {firstRun ? "稍后设置" : "取消"}
            </Button>
            <Button
              variant="primary"
              disabled={busy || !draft.provider}
              onClick={() => setStep("config")}
            >
              下一步
            </Button>
          </>
        ) : (
          <>
            <Button disabled={busy} onClick={onSkip ?? onClose}>
              {firstRun ? "稍后设置" : "取消"}
            </Button>
            <Button disabled={busy} onClick={() => void testConnection()}>
              {testing ? "测试中…" : "测试连接"}
            </Button>
            <Button variant="primary" disabled={busy} onClick={() => void submit()}>
              {saving ? "保存中…" : mode === "edit" ? "保存修改" : "添加模型"}
            </Button>
          </>
        )}
      </footer>
    </Modal>
  );
}

function initialDraft(providers: ProviderInfo[], model?: ModelView): Draft {
  if (model) {
    return {
      name: model.name,
      provider: model.provider,
      baseUrl: model.baseUrl,
      model: model.model,
      protocol: model.protocol,
      apiKey: "",
    };
  }
  const first = providers[0];
  return {
    name: "",
    provider: first?.id ?? "",
    baseUrl: first?.defaultBaseUrl ?? "",
    model: first?.defaultModel ?? "",
    protocol: first?.defaultProtocol ?? "",
    apiKey: "",
  };
}

function isLocalURL(raw: string) {
  try {
    const url = new URL(raw);
    const host = url.hostname.toLowerCase();
    return (
      host === "localhost" || host === "127.0.0.1" || host === "::1" || host.endsWith(".local")
    );
  } catch {
    return false;
  }
}
