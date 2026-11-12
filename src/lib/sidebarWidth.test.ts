import { beforeEach, describe, expect, it } from "vitest";
import {
  clampSidebarWidthForViewport,
  defaultSidebarWidth,
  loadPreferredSidebarWidth,
  maxSidebarWidthForViewport,
  preferredSidebarWidth,
  savePreferredSidebarWidth,
  sidebarWidthStorageKey,
} from "./sidebarWidth";

describe("sidebar width", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("keeps the preferred width within the allowed range", () => {
    expect(preferredSidebarWidth(100)).toBe(260);
    expect(preferredSidebarWidth(999)).toBe(999);
    expect(preferredSidebarWidth(Number.NaN)).toBe(defaultSidebarWidth);
  });

  it("allows the sidebar to grow to half of the viewport", () => {
    expect(maxSidebarWidthForViewport(1920)).toBe(960);
    expect(maxSidebarWidthForViewport(1200)).toBe(600);
    expect(maxSidebarWidthForViewport(900)).toBe(450);
    expect(maxSidebarWidthForViewport(700)).toBe(350);
    expect(maxSidebarWidthForViewport(400)).toBe(260);
    expect(clampSidebarWidthForViewport(900, 1200)).toBe(600);
    expect(clampSidebarWidthForViewport(480, 900)).toBe(450);
  });

  it("loads and saves the preferred width", () => {
    savePreferredSidebarWidth(400);
    expect(localStorage.getItem(sidebarWidthStorageKey)).toBe("400");
    expect(loadPreferredSidebarWidth()).toBe(400);
  });
});
