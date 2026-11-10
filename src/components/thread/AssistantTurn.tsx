import { AlertCircle, ChevronRight } from "lucide-react";
import { useMemo, useState } from "react";
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

export function CompletedTurn({
  messages,
  tools,
}: {
  messages: UIMessage[];
  tools: Map<string, ToolResultView>;
}) {
  const view = useMemo(() => buildCompletedTurn(messages, tools), [messages, tools]);
  return <AssistantTurn view={view} live={false} />;
}

export function LiveTurn({ run }: { run: RunView }) {
  const view = useMemo(() => buildLiveTurn(run), [run]);
  return <AssistantTurn view={view} live />;
}

function AssistantTurn({ view, live }: { view: AssistantTurnView; live: boolean }) {
  const { activity, answer, error, streaming, thinkingDurationMs } = view;
  const hasAnswer = answer.trim().length > 0;
  const hasActivity = activity.items.length > 0;
  const [open, setOpen] = useState(!live && !hasAnswer);
  const activityLabel =
    thinkingDurationMs !== undefined && thinkingDurationMs > 0
      ? `思考了 ${formatThoughtDuration(thinkingDurationMs)}`
      : "思考过程";

  return (
    <div
      className={`message assistant ${live ? "live" : ""}`}
      aria-live={live ? "polite" : undefined}
    >
      <div className="assistant-body">
        {live && !hasAnswer && !error && <LiveActivity activity={activity} streaming={streaming} />}
        {hasActivity && (!live || hasAnswer || Boolean(error)) && (
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
        {hasAnswer && (
          <div className={`assistant-text ${live && streaming ? "streaming" : ""}`}>
            <Markdown>{answer}</Markdown>
            {!live && <MessageActions text={answer} />}
          </div>
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
