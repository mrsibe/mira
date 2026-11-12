import { ArrowDown } from "lucide-react";
import { useMemo } from "react";
import { collectToolResults, groupMessages, sumUsage } from "../lib/activity";
import { formatCount, usageDetails } from "../lib/format";
import type { RunView } from "../lib/types";
import type { UIMessage } from "../mygo";
import { CompletedTurn, LiveTurn } from "./thread/AssistantTurn";
import { UserMessage } from "./thread/UserMessage";
import { useThreadScroll } from "./thread/useThreadScroll";

type ThreadProps = {
  messages: UIMessage[];
  run?: RunView;
  conversationId: string;
  scrollTarget?: { conversationId: string; messageIndex: number } | null;
  intelligentUIEnabled: boolean;
  onAskAgent: (message: string) => void;
  busy?: boolean;
  onEditMessage?: (message: UIMessage, content: string) => void;
  onRegenerate?: (messageIndex: number) => void;
  onFork?: (messageIndex: number) => void;
};

export function Thread({
  messages,
  run,
  conversationId,
  scrollTarget,
  intelligentUIEnabled,
  onAskAgent,
  busy = false,
  onEditMessage,
  onRegenerate,
  onFork,
}: ThreadProps) {
  const {
    scrollRef,
    spacerRef,
    nearBottom,
    onScroll,
    onWheel,
    onTouchStart,
    onKeyDown,
    jumpToBottom,
  } = useThreadScroll({ messages, run, conversationId, scrollTarget });

  const items = useMemo(() => groupMessages(messages), [messages]);
  const toolResults = useMemo(() => collectToolResults(messages), [messages]);
  const totalUsage = useMemo(() => sumUsage(messages, run?.usage), [messages, run?.usage]);
  const lastItem = items[items.length - 1];
  const lastAssistantKey = lastItem?.kind === "assistant" ? lastItem.key : null;

  return (
    <div className="thread-shell">
      <section
        ref={scrollRef}
        className="thread-scroll"
        aria-label="对话内容"
        onScroll={onScroll}
        onWheel={onWheel}
        onTouchStart={onTouchStart}
        onKeyDown={onKeyDown}
      >
        <div className="thread">
          {items.map((item) =>
            item.kind === "user" ? (
              <UserMessage
                key={item.key}
                message={item.message}
                busy={busy}
                onEdit={onEditMessage}
              />
            ) : (
              <CompletedTurn
                key={item.key}
                messages={item.messages}
                tools={toolResults}
                conversationId={conversationId}
                intelligentUIEnabled={intelligentUIEnabled}
                onAskAgent={onAskAgent}
                canRegenerate={
                  Boolean(onRegenerate) &&
                  run?.status !== "running" &&
                  !busy &&
                  item.key === lastAssistantKey
                }
                busy={busy}
                onRegenerate={onRegenerate}
                onFork={onFork}
              />
            ),
          )}
          {run && (
            <LiveTurn
              run={run}
              conversationId={conversationId}
              intelligentUIEnabled={intelligentUIEnabled}
              onAskAgent={onAskAgent}
            />
          )}
          {totalUsage.totalTokens > 0 && (
            <div className="thread-usage" title={usageDetails(totalUsage)}>
              本对话共 {formatCount(totalUsage.totalTokens)} tokens
            </div>
          )}
          <div ref={spacerRef} className="thread-anchor-spacer" aria-hidden="true" />
        </div>
      </section>
      {!nearBottom && (
        <button type="button" className="jump-bottom" onClick={jumpToBottom} title="回到底部">
          <ArrowDown size={16} />
        </button>
      )}
    </div>
  );
}
