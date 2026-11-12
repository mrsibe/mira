import { Check, Copy, GitBranch, RefreshCw } from "lucide-react";
import { useState } from "react";
import { copyText, formatTokens, usageDetails } from "../../lib/format";
import type { UsageView } from "../../mygo";

type MessageActionsProps = {
  text: string;
  usage?: UsageView;
  busy?: boolean;
  onRegenerate?: () => void;
  onFork?: () => void;
};

export function MessageActions({
  text,
  usage,
  busy = false,
  onRegenerate,
  onFork,
}: MessageActionsProps) {
  const [copied, setCopied] = useState(false);
  const hasText = text.trim().length > 0;

  return (
    <div className="message-actions">
      {hasText && (
        <button
          type="button"
          onClick={async () => {
            await copyText(text);
            setCopied(true);
            window.setTimeout(() => setCopied(false), 1400);
          }}
          title="复制回复"
        >
          {copied ? <Check size={14} /> : <Copy size={14} />}
          <span>{copied ? "已复制" : "复制"}</span>
        </button>
      )}
      {onRegenerate && (
        <button type="button" onClick={onRegenerate} disabled={busy} title="重新生成">
          <RefreshCw size={14} />
          <span>重新生成</span>
        </button>
      )}
      {onFork && (
        <button type="button" onClick={onFork} disabled={busy} title="从这里分叉新对话">
          <GitBranch size={14} />
          <span>分叉</span>
        </button>
      )}
      {usage && usage.totalTokens > 0 && (
        <span className="message-usage" title={usageDetails(usage)}>
          {formatTokens(usage.totalTokens)} tokens
        </span>
      )}
    </div>
  );
}
