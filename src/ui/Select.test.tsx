import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { Select } from "./Select";

const options = [
  { value: "system", label: "系统默认" },
  { value: "serif", label: "衬线" },
] as const;

describe("Select", () => {
  it("opens the menu, selects an option, and restores focus", () => {
    const onChange = vi.fn();
    render(<Select label="界面字体" value="system" options={options} onChange={onChange} />);

    const trigger = screen.getByRole("button", { name: "界面字体" });
    fireEvent.click(trigger);
    expect(screen.getByRole("menu", { name: "界面字体" })).toBeTruthy();

    fireEvent.click(screen.getByRole("menuitemradio", { name: "衬线" }));
    expect(onChange).toHaveBeenCalledWith("serif");
    expect(screen.queryByRole("menu", { name: "界面字体" })).toBeNull();
    expect(document.activeElement).toBe(trigger);
  });

  it("opens with ArrowDown and closes with Escape", () => {
    render(<Select label="代码字号" value="system" options={options} onChange={vi.fn()} />);

    const trigger = screen.getByRole("button", { name: "代码字号" });
    fireEvent.keyDown(trigger, { key: "ArrowDown" });
    expect(screen.getByRole("menu", { name: "代码字号" })).toBeTruthy();

    fireEvent.keyDown(screen.getByRole("menu", { name: "代码字号" }), { key: "Escape" });
    expect(screen.queryByRole("menu", { name: "代码字号" })).toBeNull();
    expect(document.activeElement).toBe(trigger);
  });

  it("portals its menu inside a modal dialog", () => {
    const dialog = document.createElement("dialog");
    dialog.setAttribute("open", "");
    document.body.append(dialog);

    render(<Select label="界面字体" value="system" options={options} onChange={vi.fn()} />, {
      container: dialog,
    });

    fireEvent.click(screen.getByRole("button", { name: "界面字体" }));
    expect(screen.getByRole("menu", { name: "界面字体" }).closest("dialog")).toBe(dialog);
    dialog.remove();
  });
});
