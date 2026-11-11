import { describe, expect, it } from "vitest";
import { parseAssistantAnswer } from "./stream";

const rootPatch = '{"op":"add","path":"/root","value":"text-1"}';
const textPatch =
  '{"op":"add","path":"/elements/text-1","value":{"type":"Text","props":{"content":"你好","variant":"body"},"children":[]}}';

describe("parseAssistantAnswer", () => {
  it("extracts prose and a complete interface spec", () => {
    const parsed = parseAssistantAnswer(
      `这是回答。\n\n\`\`\`spec\n${rootPatch}\n${textPatch}\n\`\`\``,
    );

    expect(parsed.text).toBe("这是回答。");
    expect(parsed.spec?.status).toBe("complete");
    expect(parsed.spec?.spec.root).toBe("text-1");
    expect(parsed.spec?.spec.elements["text-1"]?.type).toBe("Text");
  });

  it("marks an unclosed interface fence as invalid after streaming", () => {
    const parsed = parseAssistantAnswer(`回答\n\`\`\`spec\n${rootPatch}`);

    expect(parsed.spec?.status).toBe("invalid");
    expect(parsed.spec?.error).toContain("没有完整结束");
    expect(parsed.text).toBe("回答");
  });

  it("keeps incomplete JSONL out of the parsed spec while streaming", () => {
    const parsed = parseAssistantAnswer(
      `回答\n\`\`\`spec\n${rootPatch}\n{"op":"add","path":"/elements`,
      { streaming: true },
    );

    expect(parsed.spec?.status).toBe("streaming");
    expect(parsed.spec?.spec.root).toBe("text-1");
  });

  it("rejects malformed patches without exposing the fence", () => {
    const parsed = parseAssistantAnswer(`回答\n\`\`\`spec\nnot-json\n\`\`\``);

    expect(parsed.text).toBe("回答");
    expect(parsed.spec?.status).toBe("invalid");
    expect(parsed.spec?.error).toContain("无效");
  });

  it("returns plain Markdown unchanged when there is no interface", () => {
    expect(parseAssistantAnswer("普通回答")).toEqual({
      text: "普通回答",
      spec: null,
      segments: [{ kind: "text", key: "text-0", text: "普通回答" }],
    });
  });

  it("does not treat a normal Markdown language fence as a UI spec", () => {
    expect(parseAssistantAnswer("```spec-notes\n示例\n```")).toEqual({
      text: "```spec-notes\n示例\n```",
      spec: null,
      segments: [{ kind: "text", key: "text-0", text: "```spec-notes\n示例\n```" }],
    });
  });

  it("preserves the interface position between prose sections", () => {
    const parsed = parseAssistantAnswer(
      `开头回答。\n\n\`\`\`spec\n${rootPatch}\n${textPatch}\n\`\`\`\n\n结尾回答。`,
    );

    expect(parsed.text).toBe("开头回答。\n\n结尾回答。");
    expect(parsed.segments.map((segment) => segment.kind)).toEqual(["text", "spec", "text"]);
    expect(parsed.segments[0]).toEqual({ kind: "text", key: "text-0", text: "开头回答。" });
    expect(parsed.segments[2]).toEqual({ kind: "text", key: "text-2", text: "结尾回答。" });
  });

  it("reuses a completed spec while later prose streams", () => {
    const interfaceText = `开头回答。\n\n\`\`\`spec\n${rootPatch}\n${textPatch}\n\`\`\``;
    const first = parseAssistantAnswer(interfaceText, { streaming: true });
    const next = parseAssistantAnswer(`${interfaceText}\n\n后续文字`, { streaming: true });

    expect(first.spec?.status).toBe("complete");
    expect(next.spec).toBe(first.spec);
    expect(next.text).toBe("开头回答。\n\n后续文字");
  });
});
