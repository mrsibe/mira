import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { defaultAppearance } from "../../lib/appearance";
import { GeneralSettings } from "./GeneralSettings";

describe("GeneralSettings", () => {
  it("resets only appearance preferences", () => {
    const onAppearanceChange = vi.fn();
    const onThemeChange = vi.fn();

    render(
      <GeneralSettings
        theme="dark"
        onThemeChange={onThemeChange}
        appearance={{
          ...defaultAppearance,
          interfaceFont: "serif",
          chatSize: "xlarge",
        }}
        systemFonts={["Aptos", "Noto Sans SC"]}
        onAppearanceChange={onAppearanceChange}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "恢复默认" }));

    expect(onAppearanceChange).toHaveBeenCalledWith(defaultAppearance);
    expect(onThemeChange).toHaveBeenCalledWith("system");
  });

  it("selects an installed system font from the dropdown", () => {
    const onAppearanceChange = vi.fn();

    render(
      <GeneralSettings
        theme="system"
        onThemeChange={vi.fn()}
        appearance={defaultAppearance}
        systemFonts={["Aptos", "Noto Sans SC"]}
        onAppearanceChange={onAppearanceChange}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "对话字体" }));
    expect(screen.getByText("系统字体")).toBeTruthy();
    fireEvent.click(screen.getByRole("menuitemradio", { name: "Noto Sans SC" }));

    expect(onAppearanceChange).toHaveBeenCalledWith(
      expect.objectContaining({ chatFont: "family:Noto Sans SC" }),
    );
  });

  it("uses numeric size labels without semantic prefixes", () => {
    render(
      <GeneralSettings
        theme="system"
        onThemeChange={vi.fn()}
        appearance={defaultAppearance}
        systemFonts={[]}
        onAppearanceChange={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "对话字号" }));
    expect(screen.getByRole("menuitemradio", { name: "18 px" })).toBeTruthy();
    expect(screen.queryByText(/标准/)).toBeNull();
  });
});
