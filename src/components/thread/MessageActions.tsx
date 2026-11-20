import { Check, Copy, GitBranch, RefreshCw } from "lucide-react";
import { useState } from "react";
import { useMessages } from "../../i18n";
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
  const messages = useMessages();
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
          title={messages.threadCopyReply()}
        >
          {copied ? <Check size={14} /> : <Copy size={14} />}
          <span>{copied ? messages.commonCopied() : messages.commonCopy()}</span>
        </button>
      )}
      {onRegenerate && (
        <button
          type="button"
          onClick={onRegenerate}
          disabled={busy}
          title={messages.threadRegenerate()}
        >
          <RefreshCw size={14} />
          <span>{messages.threadRegenerate()}</span>
        </button>
      )}
      {onFork && (
        <button type="button" onClick={onFork} disabled={busy} title={messages.threadForkTitle()}>
          <GitBranch size={14} />
          <span>{messages.threadFork()}</span>
        </button>
      )}
      {usage && usage.totalTokens > 0 && (
        <span className="message-usage" title={usageDetails(usage, messages)}>
          {formatTokens(usage.totalTokens)} tokens
        </span>
      )}
    </div>
  );
}
