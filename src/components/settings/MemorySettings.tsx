import { RefreshCw, Trash2 } from "lucide-react";
import { useState } from "react";
import { useLocale, useMessages } from "../../i18n";
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
  const t = useMessages();
  const locale = useLocale();
  const [confirmClear, setConfirmClear] = useState(false);

  return (
    <>
      <section className="settings-switches">
        <Switch
          checked={draft.memoryEnabled}
          title={t.memoryEnableTitle()}
          description={t.memoryEnableDescription()}
          onChange={() => setField("memoryEnabled", !draft.memoryEnabled)}
        />
        {draft.memoryEnabled && (
          <>
            <Switch
              checked={draft.memorySummaryEnabled}
              title={t.memorySummaryTitle()}
              description={t.memorySummaryDescription()}
              onChange={() => setField("memorySummaryEnabled", !draft.memorySummaryEnabled)}
            />
            <Switch
              checked={draft.memorySearchEnabled}
              title={t.memorySearchTitle()}
              description={t.memorySearchDescription()}
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
                  {memoryStatusTitle(editor.memory, editor.action, t)}
                  {memoryStatusDetail(editor.memory, locale, t) &&
                    ` · ${memoryStatusDetail(editor.memory, locale, t)}`}
                </span>
                <Button
                  variant="quiet"
                  disabled={editor.action || editor.loading}
                  onClick={() => void editor.regenerate()}
                >
                  <RefreshCw size={14} className={editor.action ? "spin" : undefined} />
                  {editor.action ? t.memoryUpdating() : t.memoryUpdate()}
                </Button>
              </div>
              {editor.memory?.lastError && (
                <p className="form-error">{t.memoryUpdateFailed(editor.memory.lastError)}</p>
              )}
              <Field label={t.memoryCurrentSummary()}>
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
                    placeholder={t.memorySummaryPlaceholder()}
                  />
                )}
              </Field>
            </>
          )}

          {editor.conflict && (
            <p className="settings-description">
              {t.memoryChangedInBackground()}
              <button type="button" className="inline-link" onClick={() => void editor.reload()}>
                {t.memoryLoadLatest()}
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
            {t.memoryClear()}
          </Button>
        </section>
      )}

      {confirmClear && (
        <ConfirmDialog
          title={t.memoryClearTitle()}
          message={t.memoryClearMessage()}
          confirmLabel={t.memoryClearConfirm()}
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

function memoryStatusTitle(
  memory: MemoryView | null,
  action: boolean,
  t: ReturnType<typeof useMessages>,
) {
  if (action || memory?.running) return t.memoryStatusUpdating();
  if (!memory) return t.memoryStatusReading();
  if (memory.pendingUserTurns >= memory.summaryThreshold) return t.memoryStatusPending();
  return memory.summary ? t.memoryStatusReady() : t.memoryStatusEmpty();
}

function memoryStatusDetail(
  memory: MemoryView | null,
  locale: string,
  t: ReturnType<typeof useMessages>,
) {
  if (!memory) return "";
  if (memory.pendingUserTurns > 0) {
    return t.memoryStatusPendingDetail(
      Math.min(memory.pendingUserTurns, memory.summaryThreshold),
      memory.summaryThreshold,
    );
  }
  return memory.updatedAt ? t.memoryStatusUpdatedAt(formatTimestamp(memory.updatedAt, locale)) : "";
}

function formatTimestamp(value: string, locale: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat(locale, {
    month: "numeric",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}
