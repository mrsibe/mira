import { Check, ChevronDown, ChevronUp, Copy, Pencil } from "lucide-react";
import { useEffect, useLayoutEffect, useRef, useState } from "react";
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
              aria-label="编辑消息"
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
                title="取消编辑"
              >
                取消
              </button>
              <button
                type="button"
                className="user-editor-save"
                disabled={!draft.trim() || busy}
                onClick={submitEditing}
                title="保存并重新生成后续回答"
              >
                保存并重新生成
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
                  {expanded ? "收起" : "显示更多"}
                </button>
              )}
            </div>
            <div className="user-message-actions">
              <button
                type="button"
                title="复制消息"
                onClick={async () => {
                  await copyText(text);
                  setCopied(true);
                  window.setTimeout(() => setCopied(false), 1400);
                }}
              >
                {copied ? <Check size={14} /> : <Copy size={14} />}
                <span>{copied ? "已复制" : "复制"}</span>
              </button>
              {onEdit && (
                <button type="button" disabled={busy} onClick={beginEditing} title="编辑消息">
                  <Pencil size={14} />
                  <span>编辑</span>
                </button>
              )}
            </div>
          </>
        )}
      </div>
    </div>
  );
}
