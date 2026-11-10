import { RefreshCw, Trash2 } from "lucide-react";
import { useState } from "react";
import type { MemoryView } from "../../mygo";
import { Button, Field, Switch } from "../../ui";
import { ConfirmDialog } from "../ConfirmDialog";
import type { SetDraftField, SettingsDraft } from "./types";
import type { MemoryEditor } from "./useMemoryEditor";

type MemorySettingsProps = {
  draft: SettingsDraft;
  setField: SetDraftField;
  editor: MemoryEditor;
};

export function MemorySettings({ draft, setField, editor }: MemorySettingsProps) {
  const [confirmClear, setConfirmClear] = useState(false);

  return (
    <>
      <section className="settings-group">
        <h3 className="settings-group-title">长期记忆</h3>
        <p className="settings-description">保存在本机，不会同步。</p>
        <div className="settings-switches">
          <Switch
            checked={draft.memoryEnabled}
            title="启用"
            description="允许 Mira 参考长期信息和历史对话。"
            onChange={() => setField("memoryEnabled", !draft.memoryEnabled)}
          />
          {draft.memoryEnabled && (
            <>
              <Switch
                checked={draft.memorySummaryEnabled}
                title="自动摘要"
                description="每累计 8 条用户消息整理一次。"
                onChange={() => setField("memorySummaryEnabled", !draft.memorySummaryEnabled)}
              />
              <Switch
                checked={draft.memorySearchEnabled}
                title="历史搜索"
                description="需要时搜索其他会话。"
                onChange={() => setField("memorySearchEnabled", !draft.memorySearchEnabled)}
              />
            </>
          )}
        </div>
      </section>

      {draft.memoryEnabled && (
        <>
          {draft.memorySummaryEnabled && (
            <section className="settings-group">
              <div className="memory-status">
                <div>
                  <strong>{memoryStatusTitle(editor.memory, editor.action)}</strong>
                  <span>{memoryStatusDetail(editor.memory)}</span>
                </div>
                <Button
                  disabled={editor.action || editor.loading}
                  onClick={() => void editor.regenerate()}
                >
                  <RefreshCw size={14} className={editor.action ? "spin" : undefined} />
                  {editor.action ? "更新中…" : "更新摘要"}
                </Button>
              </div>
              {editor.memory?.lastError && (
                <p className="form-error">摘要更新失败：{editor.memory.lastError}</p>
              )}
            </section>
          )}

          <section className="settings-group">
            <Field label="手动记忆">
              {(id) => (
                <textarea
                  id={id}
                  className="settings-textarea"
                  value={editor.manualDraft}
                  maxLength={4000}
                  rows={5}
                  disabled={!editor.memory || editor.loading}
                  onChange={(event) => {
                    editor.setManualDraft(event.target.value);
                    editor.markDirty();
                  }}
                  placeholder="写入希望 Mira 长期记住的稳定信息。"
                />
              )}
            </Field>
            <span className="field-meta">{editor.manualDraft.length} / 4000</span>

            {draft.memorySummaryEnabled && (
              <>
                <Field label="自动摘要">
                  {(id) => (
                    <textarea
                      id={id}
                      className="settings-textarea"
                      value={editor.summaryDraft}
                      maxLength={2000}
                      rows={6}
                      disabled={!editor.memory || editor.loading}
                      onChange={(event) => {
                        editor.setSummaryDraft(event.target.value);
                        editor.markDirty();
                      }}
                      placeholder="Mira 会根据最近的对话自动整理。"
                    />
                  )}
                </Field>
                <span className="field-meta">{editor.summaryDraft.length} / 2000</span>
              </>
            )}

            {editor.conflict && (
              <p className="settings-description">
                记忆已在后台更新。请先复制未保存的修改，再
                <button type="button" className="inline-link" onClick={() => void editor.reload()}>
                  载入最新记忆
                </button>
                。
              </p>
            )}

            <Button
              variant="dangerText"
              className="settings-clear-memory"
              disabled={editor.action || editor.loading || !editor.memory}
              onClick={() => setConfirmClear(true)}
            >
              <Trash2 size={14} />
              清空全部记忆
            </Button>
          </section>
        </>
      )}

      {confirmClear && (
        <ConfirmDialog
          title="清空全部记忆？"
          message="手动记忆和自动摘要都会被删除，下一次自动更新会从空的记忆开始。"
          confirmLabel="清空"
          onConfirm={() => {
            setConfirmClear(false);
            void editor.clear();
          }}
          onClose={() => setConfirmClear(false)}
        />
      )}
    </>
  );
}

function memoryStatusTitle(memory: MemoryView | null, action: boolean) {
  if (action || memory?.running) return "正在更新摘要…";
  if (!memory) return "正在读取记忆…";
  if (memory.pendingUserTurns >= memory.summaryThreshold) return "等待后台更新";
  return memory.summary ? "摘要已就绪" : "尚未生成摘要";
}

function memoryStatusDetail(memory: MemoryView | null) {
  if (!memory) return "";
  const parts: string[] = [];
  if (memory.updatedAt) {
    parts.push(`更新于 ${formatTimestamp(memory.updatedAt)}`);
  } else {
    parts.push("尚未生成自动摘要");
  }
  if (memory.pendingUserTurns > 0) {
    parts.push(
      `待处理 ${Math.min(memory.pendingUserTurns, memory.summaryThreshold)} / ${memory.summaryThreshold} 条消息`,
    );
  } else {
    parts.push("没有待处理的对话");
  }
  return parts.join(" · ");
}

function formatTimestamp(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat("zh-CN", {
    month: "numeric",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}
