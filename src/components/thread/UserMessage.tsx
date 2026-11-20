import { Check, ChevronDown, ChevronUp, Copy, Pencil } from "lucide-react";
import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { useMessages } from "../../i18n";
import { messageText } from "../../lib/activity";
import { copyText } from "../../lib/format";
import type { UIMessage } from "../../mygo";

const collapsedHeight = 320;

type UserMessageProps = {
  message: UIMessage;
  busy?: boolean;
  onEdit?: (message: UIMessage, content: string) => void;
};

export function UserMessage({ message, busy = false, onEdit }: UserMessageProps) {
  const t = useMessages();
  const bubbleRef = useRef<HTMLDivElement>(null);
  const editorRef = useRef<HTMLTextAreaElement>(null);
  const [expanded, setExpanded] = useState(false);
  const [overflowing, setOverflowing] = useState(false);
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState("");
  const [copied, setCopied] = useState(false);
  const text = messageText(message);

  // biome-ignore lint/correctness/useExhaustiveDependencies: rerun measurement when the text changes
  useLayoutEffect(() => {
    const bubble = bubbleRef.current;
    if (!bubble) return;

    const measure = () => {
      setOverflowing(bubble.scrollHeight > collapsedHeight + 1);
    };
    measure();

    if (typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(measure);
    observer.observe(bubble);
    return () => observer.disconnect();
  }, [text]);

  useEffect(() => {
    if (!editing) setDraft(text);
  }, [editing, text]);

  useEffect(() => {
    if (!editing) return;
    const editor = editorRef.current;
    if (!editor) return;
    editor.focus();
    editor.setSelectionRange(editor.value.length, editor.value.length);
  }, [editing]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: draft changes need a height remeasure
  useLayoutEffect(() => {
    if (!editing) return;
    const editor = editorRef.current;
    if (!editor) return;
    editor.style.height = "0px";
    editor.style.height = `${Math.min(editor.scrollHeight, 320)}px`;
  }, [draft, editing]);

  function beginEditing() {
    if (busy || !onEdit) return;
    setDraft(text);
    setEditing(true);
  }

  function cancelEditing() {
    setDraft(text);
    setEditing(false);
  }

  function submitEditing() {
    const next = draft.trim();
    if (!next || busy) return;
    setEditing(false);
    if (next !== text) onEdit?.(message, next);
  }

  return (
    <div className="message user" data-message-index={message.index}>
      <div className="user-message">
        {editing ? (
          <div className="user-editor">
            <textarea
              ref={editorRef}
              value={draft}
              rows={1}
              aria-label={t.threadEditMessage()}
              onChange={(event) => setDraft(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === "Escape") {
                  event.preventDefault();
                  cancelEditing();
                  return;
                }
                if (event.key === "Enter" && !event.shiftKey && !event.nativeEvent.isComposing) {
                  event.preventDefault();
                  submitEditing();
                }
              }}
            />
            <div className="user-editor-actions">
              <button
                type="button"
                className="user-editor-cancel"
                onClick={cancelEditing}
                title={t.threadCancelEdit()}
              >
                {t.commonCancel()}
              </button>
              <button
                type="button"
                className="user-editor-save"
                disabled={!draft.trim() || busy}
                onClick={submitEditing}
                title={t.threadSaveAndRegenerate()}
              >
                {t.threadSaveAndRegenerateShort()}
              </button>
            </div>
          </div>
        ) : (
          <>
            <div
              ref={bubbleRef}
              className={`user-bubble ${overflowing && !expanded ? "collapsed" : ""}`}
            >
              <div className="user-bubble-text">{text}</div>
              {overflowing && (
                <button
                  type="button"
                  className="user-bubble-toggle"
                  aria-expanded={expanded}
                  onClick={() => setExpanded((value) => !value)}
                >
                  {expanded ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
                  {expanded ? t.threadCollapse() : t.threadShowMore()}
                </button>
              )}
            </div>
            <div className="user-message-actions">
              <button
                type="button"
                title={t.threadCopyMessage()}
                onClick={async () => {
                  await copyText(text);
                  setCopied(true);
                  window.setTimeout(() => setCopied(false), 1400);
                }}
              >
                {copied ? <Check size={14} /> : <Copy size={14} />}
                <span>{copied ? t.commonCopied() : t.commonCopy()}</span>
              </button>
              {onEdit && (
                <button
                  type="button"
                  disabled={busy}
                  onClick={beginEditing}
                  title={t.threadEditMessage()}
                >
                  <Pencil size={14} />
                  <span>{t.commonEdit()}</span>
                </button>
              )}
            </div>
          </>
        )}
      </div>
    </div>
  );
}
