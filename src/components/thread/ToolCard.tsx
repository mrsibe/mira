import { Wrench } from "lucide-react";
import { useState } from "react";
import { useMessages } from "../../i18n";
import { toolActivityLabel } from "../../lib/activity";
import type { ToolRun } from "../../lib/types";

export function ToolCard({ tool }: { tool: ToolRun }) {
  const t = useMessages();
  const [open, setOpen] = useState(tool.status === "running");

  return (
    <details
      className={`tool-card ${tool.isError ? "has-error" : ""}`}
      open={open}
      onToggle={(event) => setOpen(event.currentTarget.open)}
    >
      <summary>
        <span className={`tool-dot ${tool.status}`} />
        <Wrench size={14} />
        <strong>{toolActivityLabel(tool, t)}</strong>
      </summary>
      <div className="tool-detail">
        {tool.arguments && (
          <>
            <span>{t.threadToolParams()}</span>
            <pre>{tool.arguments}</pre>
          </>
        )}
        {tool.result !== undefined && (
          <>
            <span>{tool.isError ? t.threadToolError() : t.threadToolResult()}</span>
            <pre>{tool.result}</pre>
          </>
        )}
      </div>
    </details>
  );
}
