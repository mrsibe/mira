import type { StateModel } from "@json-render/core";

const prefix = "mira:genui:v1:";
const maxStoredBytes = 64 * 1024;

export function uiStateKey(conversationId: string | undefined, messageIndex: number | undefined) {
  if (!conversationId || messageIndex === undefined) return "";
  return `${prefix}${conversationId}:${messageIndex}`;
}

export function loadUIState(key: string, initial: StateModel): StateModel {
  if (!key) return initial;
  try {
    const raw = localStorage.getItem(key);
    if (!raw) return initial;
    const saved: unknown = JSON.parse(raw);
    if (!isRecord(saved) || saved.version !== 1 || !isRecord(saved.state)) return initial;
    return { ...initial, ...saved.state };
  } catch {
    return initial;
  }
}

export function saveUIState(key: string, state: StateModel) {
  if (!key) return;
  try {
    const payload = JSON.stringify({ version: 1, state });
    if (payload.length > maxStoredBytes) return;
    localStorage.setItem(key, payload);
  } catch {
    // State persistence is an enhancement; the live UI keeps working without it.
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}
