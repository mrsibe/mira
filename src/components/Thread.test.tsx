import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { UIMessage } from "../mygo";
import { Thread } from "./Thread";
import { anchorSpacerHeight } from "./thread/useThreadScroll";

const messages: UIMessage[] = [
  {
    index: 1,
    role: "assistant",
    blocks: [
      {
        type: "text",
        text: [
          "回答正文",
          "```spec",
          '{"op":"add","path":"/root","value":"text-1"}',
          '{"op":"add","path":"/elements/text-1","value":{"type":"Text","props":{"content":"不应显示","variant":"body"},"children":[]}}',
          "```",
        ].join("\n"),
      },
    ],
  },
];

const inlineMessages: UIMessage[] = [
  {
    index: 1,
    role: "assistant",
    blocks: [
      {
        type: "text",
        text: [
          "开头回答",
          "```spec",
          '{"op":"add","path":"/root","value":"text-1"}',
          '{"op":"add","path":"/elements/text-1","value":{"type":"Text","props":{"content":"界面内容","variant":"body"},"children":[]}}',
          "```",
          "结尾回答",
        ].join("\n"),
      },
    ],
  },
];

describe("Thread", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("hides the interface and spec fence when intelligent UI is disabled", () => {
    render(
      <Thread
        messages={messages}
        conversationId="conversation-1"
        intelligentUIEnabled={false}
        onAskAgent={vi.fn()}
      />,
    );

    expect(screen.getByText("回答正文")).toBeTruthy();
    expect(screen.queryByText("不应显示")).toBeNull();
    expect(screen.queryByText(/op.*add/)).toBeNull();
  });

  it("renders an interface at its position between prose sections", async () => {
    render(
      <Thread
        messages={inlineMessages}
        conversationId="conversation-1"
        intelligentUIEnabled
        onAskAgent={vi.fn()}
      />,
    );

    const intro = screen.getByText("开头回答");
    const ui = await screen.findByText("界面内容");
    const ending = screen.getByText("结尾回答");
    expect(intro.compareDocumentPosition(ui) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(ui.compareDocumentPosition(ending) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it("reserves only the space that is actually missing below the target", () => {
    expect(anchorSpacerHeight(600, 1000, 700)).toBe(300);
    expect(anchorSpacerHeight(600, 1000, 0)).toBe(0);
    expect(anchorSpacerHeight(600, 600, 0)).toBe(0);
  });

  it("aligns a new message without snapping to the bottom when the user takes control", () => {
    let contentHeight = 1000;
    const spacerHeight = (element: HTMLElement) =>
      Number.parseFloat(
        element.querySelector<HTMLElement>(".thread-anchor-spacer")?.style.height ?? "0",
      ) || 0;

    vi.spyOn(HTMLElement.prototype, "clientHeight", "get").mockImplementation(function (
      this: HTMLElement,
    ) {
      return this.classList.contains("thread-scroll") ? 600 : 100;
    });
    vi.spyOn(HTMLElement.prototype, "scrollHeight", "get").mockImplementation(function (
      this: HTMLElement,
    ) {
      return this.classList.contains("thread-scroll") ? contentHeight + spacerHeight(this) : 100;
    });
    vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockImplementation(function (
      this: HTMLElement,
    ) {
      const top = this.classList.contains("thread-scroll")
        ? 0
        : Number(this.closest<HTMLElement>("[data-message-index]")?.dataset.messageIndex) === 2
          ? 700
          : 0;
      return new DOMRect(0, top, 600, top === 700 ? 80 : 600);
    });

    const target = { conversationId: "conversation-1", messageIndex: 2 };
    const { container, rerender } = render(
      <Thread
        messages={[
          ...messages,
          {
            index: 2,
            role: "user",
            blocks: [{ type: "text", text: "新消息" }],
          },
        ]}
        conversationId="conversation-1"
        scrollTarget={target}
        intelligentUIEnabled={false}
        onAskAgent={vi.fn()}
      />,
    );

    const scroll = container.querySelector<HTMLElement>(".thread-scroll");
    const spacer = container.querySelector<HTMLElement>(".thread-anchor-spacer");
    if (!scroll) throw new Error("missing scroll container");
    if (!spacer) throw new Error("missing anchor spacer");
    expect(scroll?.scrollTop).toBe(700);
    expect(spacer?.style.height).toBe("300px");

    fireEvent.wheel(scroll as HTMLElement, { deltaY: 120 });
    scroll.scrollTop = 650;
    fireEvent.scroll(scroll);

    expect(scroll.scrollTop).toBe(650);
    expect(spacer.style.height).toBe("300px");

    contentHeight = 1400;
    rerender(
      <Thread
        messages={[
          ...messages,
          {
            index: 2,
            role: "user",
            blocks: [{ type: "text", text: "新消息" }],
          },
          {
            index: 3,
            role: "assistant",
            blocks: [{ type: "text", text: "流式回答" }],
          },
        ]}
        conversationId="conversation-1"
        scrollTarget={target}
        intelligentUIEnabled={false}
        onAskAgent={vi.fn()}
      />,
    );

    expect(scroll?.scrollTop).toBe(650);
    expect(spacer?.style.height).toBe("0px");
  });
});
