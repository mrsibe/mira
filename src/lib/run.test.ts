import { describe, expect, it, vi } from "vitest";
import { applyChatEvent, emptyRun } from "./run";

describe("applyChatEvent", () => {
  it("merges streamed text into one answer part", () => {
    vi.spyOn(Date, "now").mockReturnValue(1_000);
    let run = emptyRun();
    run = applyChatEvent(run, { type: "message_delta", text: "你" });
    run = applyChatEvent(run, { type: "message_delta", text: "好" });

    expect(run.parts).toEqual([{ id: "text-0", kind: "text", text: "你好" }]);
    expect(run.thinkingEndedAt).toBe(1_000);
  });

  it("preserves chronological ordering across thinking, tools and answer", () => {
    let run = emptyRun();
    run = applyChatEvent(run, { type: "thinking_start" });
    run = applyChatEvent(run, { type: "thinking", text: "分析问题" });
    run = applyChatEvent(run, { type: "thinking_end" });
    run = applyChatEvent(run, {
      type: "tool_start",
      toolCall: { id: "call-1", name: "web_search", arguments: '{"query":"mira"}' },
    });
    run = applyChatEvent(run, {
      type: "tool_end",
      toolResult: { id: "call-1", name: "web_search", text: "结果" },
    });
    run = applyChatEvent(run, { type: "message_delta", text: "最终回答" });

    expect(run.parts.map((part) => part.kind)).toEqual(["thinking", "tool", "text"]);
    expect(run.parts[0]).toMatchObject({ text: "分析问题", active: false });
    expect(run.parts[1]).toMatchObject({
      tool: { id: "call-1", status: "done", result: "结果" },
    });
    expect(run.parts[2]).toMatchObject({ text: "最终回答" });
  });

  it("reuses an existing tool part when a delayed start arrives", () => {
    let run = applyChatEvent(undefined, {
      type: "tool_end",
      toolResult: { id: "call-2", name: "search_memory", text: "旧结果" },
    });
    run = applyChatEvent(run, {
      type: "tool_start",
      toolCall: { id: "call-2", name: "search_memory", arguments: "{}" },
    });

    expect(run.parts).toHaveLength(1);
    expect(run.parts[0]).toMatchObject({
      kind: "tool",
      tool: { id: "call-2", status: "running", arguments: "{}" },
    });
  });

  it("records agent errors without discarding existing parts", () => {
    let run = applyChatEvent(undefined, { type: "message_delta", text: "部分回答" });
    run = applyChatEvent(run, {
      type: "error",
      error: "模型连接中断",
    });

    expect(run.status).toBe("error");
    expect(run.error).toBe("模型连接中断");
    expect(run.parts).toHaveLength(1);
  });
});
