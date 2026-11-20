import { ArrowLeft, Eye, EyeOff, KeyRound } from "lucide-react";
import { useId, useState } from "react";
import { useMessages } from "../i18n";
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
  const t = useMessages();
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
    if (!provider) return t.wizardProviderRequired();
    if (!draft.model.trim()) {
      return t.wizardFieldRequired(provider.modelLabel || t.wizardModelFallback());
    }
    if (provider.requiresBaseUrl && !draft.baseUrl.trim()) return t.wizardBaseUrlRequired();
    const local = isLocalURL(draft.baseUrl);
    const hasUsableKey = draft.apiKey.trim() !== "" || hasKey || keyStorage === "environment";
    if (provider.requiresApiKey && !(provider.localAuthOptional && local) && !hasUsableKey) {
      return t.wizardApiKeyRequired();
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
      setNotice(t.wizardConnected());
    } catch (testError) {
      setError(testError instanceof Error ? testError.message : t.wizardConnectionFailed());
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
      setError(submitError instanceof Error ? submitError.message : t.wizardSaveModelFailed());
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
      setNotice(t.wizardKeyCleared());
    } catch (clearError) {
      setError(clearError instanceof Error ? clearError.message : t.wizardClearKeyFailed());
    } finally {
      setClearing(false);
    }
  }

  const title =
    mode === "edit" ? t.wizardEditTitle() : firstRun ? t.wizardFirstTitle() : t.wizardAddTitle();

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
              ? t.wizardProviderStepDescription()
              : mode === "edit"
                ? t.wizardEditDescription()
                : t.wizardAddDescription()}
          </p>
        </div>
        <span className="model-wizard-step">{step === "provider" ? "1 / 2" : "2 / 2"}</span>
      </header>

      <div className="model-wizard-body">
        {step === "provider" ? (
          <div className="model-wizard-provider-select">
            <Field label={t.wizardProvider()}>
              {(id) => (
                <Select
                  id={id}
                  className="settings-select"
                  label={t.wizardProvider()}
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
                <span className="settings-label">{t.wizardProvider()}</span>
                <strong>{provider?.label ?? draft.provider}</strong>
              </div>
              {mode === "add" && (
                <Button disabled={busy} onClick={() => setStep("provider")}>
                  <ArrowLeft size={15} />
                  {t.wizardChange()}
                </Button>
              )}
            </div>

            <Field label={t.wizardNameOptional()}>
              {(id) => (
                <Input
                  id={id}
                  value={draft.name}
                  placeholder={draft.model || t.wizardNamePlaceholder()}
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
              <Field label={provider?.modelLabel || t.wizardModelFallback()}>
                {(id) => (
                  <Input
                    id={id}
                    value={draft.model}
                    placeholder={provider?.defaultModel || t.wizardModelNamePlaceholder()}
                    spellCheck={false}
                    onChange={(event) => setField("model", event.target.value)}
                  />
                )}
              </Field>
              {provider && provider.protocols.length > 1 && (
                <Field label={t.wizardApiProtocol()}>
                  {(id) => (
                    <Select
                      id={id}
                      className="settings-select"
                      label={t.wizardApiProtocol()}
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
                        ? t.wizardKeyFromEnvironment()
                        : hasKey
                          ? t.wizardKeyConfigured()
                          : provider?.localAuthOptional
                            ? t.wizardKeyLocalOptional()
                            : "sk-…"
                    }
                    spellCheck={false}
                    onChange={(event) => setField("apiKey", event.target.value)}
                  />
                  <IconButton
                    label={showKey ? t.commonHideKey() : t.commonShowKey()}
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
                {t.wizardClearSavedKey()}
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
              {firstRun ? t.wizardLater() : t.commonCancel()}
            </Button>
            <Button
              variant="primary"
              disabled={busy || !draft.provider}
              onClick={() => setStep("config")}
            >
              {t.wizardNext()}
            </Button>
          </>
        ) : (
          <>
            <Button disabled={busy} onClick={onSkip ?? onClose}>
              {firstRun ? t.wizardLater() : t.commonCancel()}
            </Button>
            <Button disabled={busy} onClick={() => void testConnection()}>
              {testing ? t.wizardTesting() : t.wizardTestConnection()}
            </Button>
            <Button variant="primary" disabled={busy} onClick={() => void submit()}>
              {saving
                ? t.commonSaving()
                : mode === "edit"
                  ? t.wizardSaveChanges()
                  : t.wizardAddModel()}
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
