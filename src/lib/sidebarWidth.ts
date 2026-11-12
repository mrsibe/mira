export const sidebarWidthStorageKey = "mira-sidebar-width";
export const defaultSidebarWidth = 320;
export const minSidebarWidth = 260;

export function preferredSidebarWidth(value: number): number {
  if (!Number.isFinite(value)) return defaultSidebarWidth;
  return Math.max(minSidebarWidth, Math.round(value));
}

export function maxSidebarWidthForViewport(viewportWidth: number): number {
  return Math.max(minSidebarWidth, Math.round(viewportWidth / 2));
}

export function clampSidebarWidthForViewport(width: number, viewportWidth: number): number {
  return Math.min(maxSidebarWidthForViewport(viewportWidth), preferredSidebarWidth(width));
}

export function loadPreferredSidebarWidth(): number {
  try {
    const raw = localStorage.getItem(sidebarWidthStorageKey);
    return raw === null ? defaultSidebarWidth : preferredSidebarWidth(Number(raw));
  } catch {
    return defaultSidebarWidth;
  }
}

export function savePreferredSidebarWidth(width: number): void {
  try {
    localStorage.setItem(sidebarWidthStorageKey, String(preferredSidebarWidth(width)));
  } catch {
    // The width still applies for this session when storage is unavailable.
  }
}

export function applySidebarWidth(width: number): void {
  document.documentElement.style.setProperty("--sidebar-width", `${width}px`);
}

export function applyStoredSidebarWidth(): void {
  applySidebarWidth(clampSidebarWidthForViewport(loadPreferredSidebarWidth(), window.innerWidth));
}
