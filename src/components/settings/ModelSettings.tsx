import { Check, Pencil, Plus, Trash2 } from "lucide-react";
import { useState } from "react";
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

const storageLabels: Record<string, string> = {
  keychain: "密钥已存入钥匙串",
  environment: "密钥来自环境变量",
  session: "密钥仅在本次会话有效",
  none: "未配置密钥",
};

export function ModelSettings({
  settings,
  onActivateModel,
  onAddModel,
  onEditModel,
  onDeleteModel,
  onError,
}: ModelSettingsProps) {
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
      onError(error instanceof Error ? error.message : "切换模型失败");
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
      onError(error instanceof Error ? error.message : "删除模型失败");
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="settings-group model-settings">
      <div>
        <h3 className="settings-group-title">模型</h3>
        <p className="settings-description">添加多个模型，并随时切换当前使用的模型。</p>
      </div>

      {models.length > 0 && (
        <Field label="当前模型">
          {(id) => (
            <Select
              id={id}
              className="settings-select"
              label="当前模型"
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
          <p>还没有模型。添加一个模型后即可开始对话。</p>
          <Button variant="primary" onClick={onAddModel}>
            <Plus size={16} />
            添加模型
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
                        当前
                      </span>
                    )}
                  </div>
                  <p>
                    {model.providerLabel} · {model.model}
                  </p>
                  <span className={`model-key-status ${model.configured ? "" : "warning"}`}>
                    {model.configured
                      ? (storageLabels[model.apiKeyStorage] ?? "已配置")
                      : "需要补充连接信息"}
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
                        确认删除
                      </Button>
                      <Button disabled={busy} onClick={() => setPendingDelete(null)}>
                        取消
                      </Button>
                    </>
                  ) : (
                    <>
                      <IconButton
                        label={`编辑 ${model.name}`}
                        disabled={busy}
                        onClick={() => onEditModel(model)}
                      >
                        <Pencil size={15} />
                      </IconButton>
                      <IconButton
                        className="model-delete"
                        label={`删除 ${model.name}`}
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
            新增模型
          </Button>
        </div>
      )}
    </section>
  );
}
