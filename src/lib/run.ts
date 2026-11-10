import type { ChatEvent } from "../mygo";
import type { RunPart, RunView } from "./types";

export function emptyRun(): RunView {
  return {
    status: "running",
    parts: [],
    nextPartId: 0,
    startedAt: Date.now(),
  };
}

export function applyChatEvent(current: RunView | undefined, event: ChatEvent): RunView {
  const run = current ?? emptyRun();
  const parts: RunPart[] = [...run.parts];
  let nextPartId = run.nextPartId;
  const next: RunView = { ...run, parts };
  const newPartId = (kind: RunPart["kind"]) => `${kind}-${nextPartId++}`;

  switch (event.type) {
    case "message_delta": {
      const text = event.text ?? "";
      if (!text) break;
      if (next.thinkingEndedAt === undefined) {
        next.thinkingEndedAt = Date.now();
      }
      const last = parts[parts.length - 1];
      if (last?.kind === "text") {
        parts[parts.length - 1] = { ...last, text: last.text + text };
      } else {
        parts.push({ id: newPartId("text"), kind: "text", text });
      }
      break;
    }
    case "thinking_start":
      next.thinkingEndedAt = undefined;
      parts.push({
        id: newPartId("thinking"),
        kind: "thinking",
        text: "",
        active: true,
      });
      break;
    case "thinking": {
      const text = event.text ?? "";
      if (!text) break;
      next.thinkingEndedAt = undefined;
      const last = parts[parts.length - 1];
      if (last?.kind === "thinking") {
        parts[parts.length - 1] = { ...last, text: last.text + text };
      } else {
        parts.push({
          id: newPartId("thinking"),
          kind: "thinking",
          text,
          active: true,
        });
      }
      break;
    }
    case "thinking_end": {
      for (let index = parts.length - 1; index >= 0; index--) {
        const part = parts[index];
        if (part?.kind !== "thinking") continue;
        parts[index] = { ...part, active: false };
        break;
      }
      break;
    }
    case "tool_start": {
      const call = event.toolCall;
      if (!call) break;
      next.thinkingEndedAt = undefined;
      const existing = parts.findIndex((part) => part.kind === "tool" && part.tool.id === call.id);
      const tool = {
        id: call.id,
        name: call.name,
        arguments: call.arguments,
        status: "running" as const,
      };
      if (existing >= 0) {
        const part = parts[existing];
        if (part?.kind === "tool") {
          parts[existing] = { ...part, tool };
        }
      } else {
        parts.push({ id: newPartId("tool"), kind: "tool", tool });
      }
      break;
    }
    case "tool_end": {
      const result = event.toolResult;
      if (!result) break;
      const existing = parts.findIndex(
        (part) => part.kind === "tool" && part.tool.id === result.id,
      );
      if (existing >= 0) {
        const part = parts[existing];
        if (part?.kind !== "tool") break;
        parts[existing] = {
          ...part,
          tool: {
            ...part.tool,
            result: result.text,
            isError: result.isError,
            status: "done",
          },
        };
      } else {
        parts.push({
          id: newPartId("tool"),
          kind: "tool",
          tool: {
            id: result.id,
            name: result.name,
            arguments: "",
            result: result.text,
            isError: result.isError,
            status: "done",
          },
        });
      }
      break;
    }
    case "error":
      next.status = "error";
      next.error = event.error ?? "生成失败";
      break;
  }

  next.nextPartId = nextPartId;
  return next;
}
