import { describe, expect, it } from "vitest";
import type { UIMessage } from "../mygo";
import { buildCompletedTurn, collectToolResults } from "./activity";

describe("buildCompletedTurn", () => {
  it("keeps thinking, tools and the final answer in chronological order", () => {
    const messages: UIMessage[] = [
      {
        index: 1,
        role: "assistant",
        blocks: [
          { type: "thinking", text: "先确认概念" },
          {
            type: "tool_call",
            toolCall: {
              id: "call-1",
              name: "web_search",
              arguments: '{"query":"Agent 双系统"}',
            },
          },
        ],
      },
      {
        index: 2,
        role: "tool",
        toolCallId: "call-1",
        toolName: "web_search",
        blocks: [{ type: "text", text: "搜索结果" }],
      },
      {
        index: 3,
        role: "assistant",
        thinkingDurationMs: 82_000,
        blocks: [{ type: "text", text: "整理后的回答" }],
      },
    ];

    const view = buildCompletedTurn(messages, collectToolResults(messages));

    expect(view.activity.items.map((item) => item.kind)).toEqual(["thinking", "tool"]);
    expect(view.activity.items[0]).toMatchObject({ text: "先确认概念" });
    expect(view.activity.items[1]).toMatchObject({
      tool: { name: "web_search", result: "搜索结果", status: "done" },
    });
    expect(view.answer).toBe("整理后的回答");
    expect(view.thinkingDurationMs).toBe(82_000);
  });
});
