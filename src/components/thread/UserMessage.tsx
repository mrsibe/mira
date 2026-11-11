import { ChevronDown, ChevronUp } from "lucide-react";
import { useLayoutEffect, useRef, useState } from "react";
import { messageText } from "../../lib/activity";
import type { UIMessage } from "../../mygo";

const collapsedHeight = 320;

export function UserMessage({ message }: { message: UIMessage }) {
  const bubbleRef = useRef<HTMLDivElement>(null);
  const [expanded, setExpanded] = useState(false);
  const [overflowing, setOverflowing] = useState(false);
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

  return (
    <div className="message user" data-message-index={message.index}>
      <div className="user-message">
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
      </div>
    </div>
  );
}
