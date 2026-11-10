import { ArrowDown } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { collectToolResults, groupMessages } from "../lib/activity";
import type { RunView } from "../lib/types";
import type { UIMessage } from "../mygo";
import { CompletedTurn, LiveTurn } from "./thread/AssistantTurn";
import { UserMessage } from "./thread/UserMessage";

type ThreadProps = {
  messages: UIMessage[];
  run?: RunView;
};

export function Thread({ messages, run }: ThreadProps) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const [nearBottom, setNearBottom] = useState(true);

  // biome-ignore lint/correctness/useExhaustiveDependencies: message and stream updates are scroll triggers
  useEffect(() => {
    const element = scrollRef.current;
    if (!element || !nearBottom) return;
    element.scrollTop = element.scrollHeight;
  }, [messages, run, nearBottom]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: reset when switching to an empty thread
  useEffect(() => {
    setNearBottom(true);
  }, [messages.length === 0]);

  const lastMessage = messages[messages.length - 1];
  // biome-ignore lint/correctness/useExhaustiveDependencies: a new user message is the scroll trigger
  useEffect(() => {
    if (lastMessage?.role !== "user") return;
    setNearBottom(true);
    const element = scrollRef.current;
    if (element) {
      window.requestAnimationFrame(() => {
        element.scrollTop = element.scrollHeight;
      });
    }
  }, [lastMessage?.index, lastMessage?.role]);

  const items = useMemo(() => groupMessages(messages), [messages]);
  const toolResults = useMemo(() => collectToolResults(messages), [messages]);

  return (
    <div className="thread-shell">
      <div
        ref={scrollRef}
        className="thread-scroll"
        onScroll={(event) => {
          const element = event.currentTarget;
          setNearBottom(element.scrollHeight - element.scrollTop - element.clientHeight < 120);
        }}
      >
        <div className="thread">
          {items.map((item) =>
            item.kind === "user" ? (
              <UserMessage key={item.key} message={item.message} />
            ) : (
              <CompletedTurn key={item.key} messages={item.messages} tools={toolResults} />
            ),
          )}
          {run && <LiveTurn run={run} />}
        </div>
      </div>
      {!nearBottom && (
        <button
          type="button"
          className="jump-bottom"
          onClick={() => {
            scrollRef.current?.scrollTo({
              top: scrollRef.current.scrollHeight,
              behavior: "smooth",
            });
            setNearBottom(true);
          }}
          title="回到底部"
        >
          <ArrowDown size={16} />
        </button>
      )}
    </div>
  );
}
