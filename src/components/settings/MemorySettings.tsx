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
      <section className="settings-switches">
        <Switch
          checked={draft.memoryEnabled}
          title="启用记忆"
          description="参考历史对话和长期信息。"
          onChange={() => setField("memoryEnabled", !draft.memoryEnabled)}
        />
        {draft.memoryEnabled && (
          <>
            <Switch
              checked={draft.memorySummaryEnabled}
              title="自动摘要"
              description="每 8 条消息整理一次。"
              onChange={() => setField("memorySummaryEnabled", !draft.memorySummaryEnabled)}
            />
            <Switch
              checked={draft.memorySearchEnabled}
              title="历史搜索"
              description="需要时检索其他对话。"
              onChange={() => setField("memorySearchEnabled", !draft.memorySearchEnabled)}
            />
          </>
        )}
      </section>

      {draft.memoryEnabled && (
        <section className="settings-group memory-editor">
          {draft.memorySummaryEnabled && (
            <>
              <div className="memory-status-row">
                <span>
                  {memoryStatusTitle(editor.memory, editor.action)}
                  {memoryStatusDetail(editor.memory) && ` · ${memoryStatusDetail(editor.memory)}`}
                </span>
                <Button
                  variant="quiet"
                  disabled={editor.action || editor.loading}
                  onClick={() => void editor.regenerate()}
                >
                  <RefreshCw size={14} className={editor.action ? "spin" : undefined} />
                  {editor.action ? "更新中…" : "更新"}
                </Button>
              </div>
              {editor.memory?.lastError && (
                <p className="form-error">摘要更新失败：{editor.memory.lastError}</p>
              )}
              <Field label="当前摘要">
                {(id) => (
                  <textarea
                    id={id}
                    className="settings-textarea"
                    value={editor.summaryDraft}
                    maxLength={2000}
                    rows={5}
                    disabled={!editor.memory || editor.loading}
                    onChange={(event) => {
                      editor.setSummaryDraft(event.target.value);
                      editor.markDirty();
                    }}
                    placeholder="由最近对话自动生成"
                  />
                )}
              </Field>
            </>
          )}

          {editor.conflict && (
            <p className="settings-description">
              记忆已在后台更新。
              <button type="button" className="inline-link" onClick={() => void editor.reload()}>
                载入最新
              </button>
            </p>
          )}

          <Button
            variant="dangerText"
            className="settings-clear-memory"
            disabled={editor.action || editor.loading || !editor.memory}
            onClick={() => setConfirmClear(true)}
          >
            <Trash2 size={14} />
            清空记忆
          </Button>
        </section>
      )}

      {confirmClear && (
        <ConfirmDialog
          title="清空记忆？"
          message="自动摘要会被删除，下一次自动更新会从空的记忆开始。"
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
  if (action || memory?.running) return "正在更新…";
  if (!memory) return "正在读取记忆…";
  if (memory.pendingUserTurns >= memory.summaryThreshold) return "等待更新";
  return memory.summary ? "摘要已就绪" : "尚未生成";
}

function memoryStatusDetail(memory: MemoryView | null) {
  if (!memory) return "";
  if (memory.pendingUserTurns > 0) {
    return `待处理 ${Math.min(memory.pendingUserTurns, memory.summaryThreshold)} / ${memory.summaryThreshold}`;
  }
  return memory.updatedAt ? `更新于 ${formatTimestamp(memory.updatedAt)}` : "";
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
