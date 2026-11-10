import { Check, Copy } from "lucide-react";
import { useState } from "react";
import { copyText } from "../../lib/format";

export function MessageActions({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);

  return (
    <div className="message-actions">
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
    </div>
  );
}
