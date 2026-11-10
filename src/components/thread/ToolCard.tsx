import { Wrench } from "lucide-react";
import { useState } from "react";
import { toolActivityLabel } from "../../lib/activity";
import type { ToolRun } from "../../lib/types";

export function ToolCard({ tool }: { tool: ToolRun }) {
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
        <strong>{toolActivityLabel(tool)}</strong>
      </summary>
      <div className="tool-detail">
        {tool.arguments && (
          <>
            <span>参数</span>
            <pre>{tool.arguments}</pre>
          </>
        )}
        {tool.result !== undefined && (
          <>
            <span>{tool.isError ? "错误" : "结果"}</span>
            <pre>{tool.result}</pre>
          </>
        )}
      </div>
    </details>
  );
}
