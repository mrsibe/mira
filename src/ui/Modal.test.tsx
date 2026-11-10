import { fireEvent, render, screen } from "@testing-library/react";
import { beforeAll, describe, expect, it, vi } from "vitest";
import { Modal } from "./Modal";

beforeAll(() => {
  HTMLDialogElement.prototype.showModal = function showModal() {
    this.open = true;
  };
  HTMLDialogElement.prototype.close = function close() {
    this.open = false;
  };
});

describe("Modal", () => {
  it("opens as a native modal dialog and closes on Escape", () => {
    const onClose = vi.fn();
    render(
      <Modal labelledBy="modal-title" onClose={onClose}>
        <h2 id="modal-title">测试弹窗</h2>
        <button type="button">确认</button>
      </Modal>,
    );

    const dialog = screen.getByRole("dialog", { name: "测试弹窗" });
    expect(dialog.hasAttribute("open")).toBe(true);

    fireEvent(dialog, new Event("cancel", { bubbles: true, cancelable: true }));
    expect(onClose).toHaveBeenCalledOnce();
  });
});
