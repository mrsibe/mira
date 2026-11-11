import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { UserMessage } from "./UserMessage";

describe("UserMessage", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("collapses long messages and expands them on request", () => {
    vi.spyOn(HTMLElement.prototype, "scrollHeight", "get").mockImplementation(function (
      this: HTMLElement,
    ) {
      return this.classList.contains("user-bubble") ? 480 : 0;
    });

    const { container } = render(
      <UserMessage
        message={{
          index: 1,
          role: "user",
          blocks: [{ type: "text", text: "很长的消息" }],
        }}
      />,
    );

    expect(container.querySelector(".user-bubble")?.classList.contains("collapsed")).toBe(true);
    expect(container.querySelector(".user-bubble .user-bubble-toggle")).not.toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "显示更多" }));
    expect(container.querySelector(".user-bubble")?.classList.contains("collapsed")).toBe(false);
    expect(screen.getByRole("button", { name: "收起" })).toBeTruthy();
  });
});
