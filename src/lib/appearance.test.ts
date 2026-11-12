import { beforeEach, describe, expect, it } from "vitest";
import {
  appearanceStorageKey,
  applyAppearance,
  buildFontOptions,
  defaultAppearance,
  familyFromChoice,
  loadAppearance,
  normalizeAppearance,
  resolveFontStack,
  saveAppearance,
} from "./appearance";

describe("appearance", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("falls back to defaults for missing or malformed storage", () => {
    expect(loadAppearance()).toEqual(defaultAppearance);
    localStorage.setItem(appearanceStorageKey, "{not-json");
    expect(loadAppearance()).toEqual(defaultAppearance);
  });

  it("normalizes unknown values and migrates legacy custom fonts", () => {
    expect(
      normalizeAppearance({
        interfaceFont: "custom",
        interfaceFontCustom: "Aptos",
        chatFont: "unknown" as never,
        chatSize: "xlarge",
        codeFont: "custom",
        codeFontCustom: "JetBrains Mono",
      }),
    ).toEqual({
      interfaceFont: "family:Aptos",
      chatFont: "system",
      chatSize: "xlarge",
      codeFont: "family:JetBrains Mono",
      codeSize: "default",
    });
  });

  it("round-trips appearance preferences", () => {
    const saved = {
      ...defaultAppearance,
      interfaceFont: "humanist" as const,
      chatFont: "family:Noto Serif SC" as const,
      chatSize: "large" as const,
      codeSize: "small" as const,
    };
    saveAppearance(saved);
    expect(loadAppearance()).toEqual(saved);
  });

  it("quotes installed family names and applies CSS variables", () => {
    const root = document.createElement("div");
    applyAppearance(
      {
        ...defaultAppearance,
        interfaceFont: 'family:My "Quoted" Font',
        chatSize: "xlarge",
        codeSize: "large",
      },
      root,
    );

    expect(root.style.getPropertyValue("--font-sans")).toContain('"My \\"Quoted\\" Font"');
    expect(root.style.getPropertyValue("--chat-font-size")).toBe("20px");
    expect(root.style.getPropertyValue("--code-font-size")).toBe("15px");
  });

  it("separates preset and installed font options", () => {
    const options = buildFontOptions("chat", ["Aptos", "Noto Sans SC"], "family:Aptos");

    expect(options[0]).toMatchObject({ value: "system", label: "系统默认" });
    expect(options.filter((option) => option.group === "系统字体")).toHaveLength(2);
    expect(options.find((option) => option.value === "family:Aptos")).toMatchObject({
      label: "Aptos",
    });
  });

  it("keeps a selected family visible before the system list loads", () => {
    const options = buildFontOptions("interface", [], "family:Fira Sans");
    expect(options.find((option) => option.value === "family:Fira Sans")).toMatchObject({
      label: "Fira Sans",
      group: "系统字体",
    });
    expect(familyFromChoice("family:Fira Sans")).toBe("Fira Sans");
    expect(resolveFontStack("family:Fira Sans", "interface")).toContain('"Fira Sans"');
  });
});
