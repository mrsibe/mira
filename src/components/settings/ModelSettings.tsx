import { Check, Pencil, Plus, Trash2 } from "lucide-react";
import { useState } from "react";
import { useMessages } from "../../i18n";
import type { ModelView, View } from "../../mygo";
import { Button, Field, IconButton, Select } from "../../ui";

type ModelSettingsProps = {
  settings: View;
  onActivateModel: (id: string) => Promise<View>;
  onAddModel: () => void;
  onEditModel: (model: ModelView) => void;
  onDeleteModel: (id: string) => Promise<View>;
  onError: (message: string) => void;
};

export function ModelSettings({
  settings,
  onActivateModel,
  onAddModel,
  onEditModel,
  onDeleteModel,
  onError,
}: ModelSettingsProps) {
  const t = useMessages();
  const storageLabels: Record<string, string> = {
    keychain: t.modelStorageKeychain(),
    environment: t.modelStorageEnvironment(),
    session: t.modelStorageSession(),
    none: t.modelStorageNone(),
  };
  const [pendingDelete, setPendingDelete] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const models = settings.models;

  async function activate(id: string) {
    if (id === settings.activeModelId || busy) return;
    setBusy(true);
    onError("");
    try {
      await onActivateModel(id);
    } catch (error) {
      onError(error instanceof Error ? error.message : t.modelActivateFailed());
    } finally {
      setBusy(false);
    }
  }

  async function remove(id: string) {
    if (busy) return;
    setBusy(true);
    onError("");
    try {
      await onDeleteModel(id);
      setPendingDelete(null);
    } catch (error) {
      onError(error instanceof Error ? error.message : t.modelDeleteFailed());
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="settings-group model-settings">
      <div>
        <h3 className="settings-group-title">{t.modelTitle()}</h3>
        <p className="settings-description">{t.modelDescription()}</p>
      </div>

      {models.length > 0 && (
        <Field label={t.modelCurrent()}>
          {(id) => (
            <Select
              id={id}
              className="settings-select"
              label={t.modelCurrent()}
              value={settings.activeModelId}
              options={models.map((model) => ({
                value: model.id,
                label: `${model.name} · ${model.providerLabel}`,
              }))}
              disabled={busy}
              onChange={(value) => void activate(value)}
            />
          )}
        </Field>
      )}

      {models.length === 0 ? (
        <div className="model-empty">
          <p>{t.modelEmpty()}</p>
          <Button variant="primary" onClick={onAddModel}>
            <Plus size={16} />
            {t.modelAdd()}
          </Button>
        </div>
      ) : (
        <div className="model-list">
          {models.map((model) => {
            const active = model.id === settings.activeModelId;
            return (
              <div key={model.id} className={`model-item ${active ? "active" : ""}`}>
                <div className="model-item-main">
                  <div className="model-item-title">
                    <span>{model.name}</span>
                    {active && (
                      <span className="model-active-badge">
                        <Check size={12} />
                        {t.modelActive()}
                      </span>
                    )}
                  </div>
                  <p>
                    {model.providerLabel} · {model.model}
                  </p>
                  <span className={`model-key-status ${model.configured ? "" : "warning"}`}>
                    {model.configured
                      ? (storageLabels[model.apiKeyStorage] ?? t.modelConfigured())
                      : t.modelNeedsConnection()}
                  </span>
                </div>
                <div className="model-item-actions">
                  {pendingDelete === model.id ? (
                    <>
                      <Button
                        variant="dangerText"
                        disabled={busy}
                        onClick={() => void remove(model.id)}
                      >
                        {t.modelConfirmDelete()}
                      </Button>
                      <Button disabled={busy} onClick={() => setPendingDelete(null)}>
                        {t.commonCancel()}
                      </Button>
                    </>
                  ) : (
                    <>
                      <IconButton
                        label={t.modelEditLabel(model.name)}
                        disabled={busy}
                        onClick={() => onEditModel(model)}
                      >
                        <Pencil size={15} />
                      </IconButton>
                      <IconButton
                        className="model-delete"
                        label={t.modelDeleteLabel(model.name)}
                        disabled={busy}
                        onClick={() => setPendingDelete(model.id)}
                      >
                        <Trash2 size={15} />
                      </IconButton>
                    </>
                  )}
                </div>
              </div>
            );
          })}
          <Button className="model-add" onClick={onAddModel}>
            <Plus size={16} />
            {t.modelAddNew()}
          </Button>
        </div>
      )}
    </section>
  );
}
