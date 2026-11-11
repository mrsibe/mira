import { describe, expect, it } from "vitest";
import catalogFile from "../../internal/prompt/ui_catalog.txt?raw";
import { genUIPrompt } from "./catalog";

describe("generated UI prompt", () => {
  it("matches the component catalog", () => {
    expect(catalogFile).toBe(`${genUIPrompt.trim()}\n`);
  });
});
