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
    expect(preferredSidebarWidth(999)).toBe(480);
    expect(preferredSidebarWidth(Number.NaN)).toBe(defaultSidebarWidth);
  });

  it("reserves space for the main content on narrow windows", () => {
    expect(maxSidebarWidthForViewport(1200)).toBe(480);
    expect(maxSidebarWidthForViewport(900)).toBe(300);
    expect(maxSidebarWidthForViewport(700)).toBe(260);
    expect(clampSidebarWidthForViewport(480, 900)).toBe(300);
  });

  it("loads and saves the preferred width", () => {
    savePreferredSidebarWidth(400);
    expect(localStorage.getItem(sidebarWidthStorageKey)).toBe("400");
    expect(loadPreferredSidebarWidth()).toBe(400);
  });
});
