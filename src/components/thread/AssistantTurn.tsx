import { AlertCircle, ChevronRight } from "lucide-react";
import { lazy, Suspense, useMemo, useState } from "react";
import { uiStateKey } from "../../genui/storage";
import {
  type AssistantTurnView,
  buildCompletedTurn,
  buildLiveTurn,
  formatThoughtDuration,
} from "../../lib/activity";
import { Markdown } from "../../lib/markdown";
import type { RunView } from "../../lib/types";
import type { ToolResultView, UIMessage } from "../../mygo";
import { ActivityContent, LiveActivity } from "./ActivityContent";
import { MessageActions } from "./MessageActions";

const GenUIRenderer = lazy(() => import("../../genui/GenUIRenderer"));

export function CompletedTurn({
  messages,
  tools,
  conversationId,
  intelligentUIEnabled,
  onAskAgent,
  canRegenerate,
  busy,
  onRegenerate,
  onFork,
}: {
  messages: UIMessage[];
  tools: Map<string, ToolResultView>;
  conversationId: string;
  intelligentUIEnabled: boolean;
  onAskAgent: (message: string) => void;
  canRegenerate: boolean;
  busy: boolean;
  onRegenerate?: (messageIndex: number) => void;
  onFork?: (messageIndex: number) => void;
}) {
  const view = useMemo(() => buildCompletedTurn(messages, tools), [messages, tools]);
  const actionIndex = messages[0]?.index;
  const completed = view.messageIndex !== undefined && actionIndex !== undefined;
  return (
    <AssistantTurn
      view={view}
      live={false}
      conversationId={conversationId}
      intelligentUIEnabled={intelligentUIEnabled}
      onAskAgent={onAskAgent}
      canRegenerate={canRegenerate}
      busy={busy}
      onRegenerate={
        completed && onRegenerate && actionIndex !== undefined
          ? () => onRegenerate(actionIndex)
          : undefined
      }
      onFork={
        completed && onFork && actionIndex !== undefined ? () => onFork(actionIndex) : undefined
      }
    />
  );
}

export function LiveTurn({
  run,
  conversationId,
  intelligentUIEnabled,
  onAskAgent,
}: {
  run: RunView;
  conversationId: string;
  intelligentUIEnabled: boolean;
  onAskAgent: (message: string) => void;
}) {
  const view = useMemo(() => buildLiveTurn(run), [run]);
  return (
    <AssistantTurn
      view={view}
      live
      conversationId={conversationId}
      intelligentUIEnabled={intelligentUIEnabled}
      onAskAgent={onAskAgent}
      canRegenerate={false}
      busy={false}
    />
  );
}

function AssistantTurn({
  view,
  live,
  conversationId,
  intelligentUIEnabled,
  onAskAgent,
  canRegenerate,
  busy,
  onRegenerate,
  onFork,
}: {
  view: AssistantTurnView;
  live: boolean;
  conversationId: string;
  intelligentUIEnabled: boolean;
  onAskAgent: (message: string) => void;
  canRegenerate: boolean;
  busy: boolean;
  onRegenerate?: () => void;
  onFork?: () => void;
}) {
  const { activity, answer, error, streaming, thinkingDurationMs, segments } = view;
  const hasAnswer = answer.trim().length > 0;
  const hasSpec = intelligentUIEnabled && segments.some((segment) => segment.kind === "spec");
  const hasContent = hasAnswer || hasSpec;
  const hasActivity = activity.items.length > 0;
  const [open, setOpen] = useState(!live && !hasContent);
  const activityLabel =
    thinkingDurationMs !== undefined && thinkingDurationMs > 0
      ? `思考了 ${formatThoughtDuration(thinkingDurationMs)}`
      : "思考过程";
  const storageKey = uiStateKey(conversationId, view.messageIndex);

  return (
    <div
      className={`message assistant ${live ? "live" : ""}`}
      aria-live={live ? "polite" : undefined}
    >
      <div className="assistant-body">
        {live && !hasContent && !error && (
          <LiveActivity activity={activity} streaming={streaming} />
        )}
        {hasActivity && (!live || hasContent || Boolean(error)) && (
          <div className="activity">
            <button
              type="button"
              className="activity-toggle"
              aria-expanded={open}
              onClick={() => setOpen((value) => !value)}
            >
              <ChevronRight size={14} className={`activity-chevron ${open ? "open" : ""}`} />
              <span>{activityLabel}</span>
            </button>
            {open && <ActivityContent activity={activity} />}
          </div>
        )}
        {segments.map((segment) =>
          segment.kind === "text" ? (
            <div
              key={segment.key}
              className={`assistant-text ${live && streaming ? "streaming" : ""}`}
            >
              <Markdown>{segment.text}</Markdown>
            </div>
          ) : intelligentUIEnabled ? (
            <GenUIBlock
              key={segment.key}
              spec={segment.spec}
              live={live}
              storageKey={storageKey}
              onAskAgent={onAskAgent}
            />
          ) : null,
        )}
        {hasContent && !live && (
          <MessageActions
            text={answer}
            usage={view.usage}
            busy={busy}
            onRegenerate={canRegenerate ? onRegenerate : undefined}
            onFork={onFork}
          />
        )}
        {error && (
          <div className="error-card">
            <AlertCircle size={17} />
            <div>
              <strong>生成失败</strong>
              <p>{error}</p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

function GenUIBlock({
  spec,
  live,
  storageKey,
  onAskAgent,
}: {
  spec: NonNullable<AssistantTurnView["spec"]>;
  live: boolean;
  storageKey: string;
  onAskAgent: (message: string) => void;
}) {
  if (spec.status === "invalid") {
    return (
      <div className="genui-failure" role="status">
        界面暂时无法显示：{spec.error ?? "界面描述不完整。"}
      </div>
    );
  }
  if (!spec.spec.root) {
    return <div className="genui-loading">正在准备界面…</div>;
  }
  return (
    <Suspense fallback={<div className="genui-loading">正在准备界面…</div>}>
      <GenUIRenderer
        spec={spec.spec}
        loading={spec.status === "streaming"}
        complete={spec.status === "complete"}
        storageKey={live ? "" : storageKey}
        onAskAgent={onAskAgent}
      />
    </Suspense>
  );
}
