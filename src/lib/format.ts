import type { Conversation } from "../mygo";

const day = 24 * 60 * 60 * 1000;

export function groupConversations(conversations: Conversation[]) {
  const now = new Date();
  const today = startOfDay(now);
  const groups = new Map<string, Conversation[]>([
    ["今天", []],
    ["昨天", []],
    ["前 7 天", []],
    ["更早", []],
  ]);

  for (const conversation of conversations) {
    const updated = new Date(conversation.updatedAt);
    const age = today.getTime() - startOfDay(updated).getTime();
    const key = age <= 0 ? "今天" : age <= day ? "昨天" : age <= day * 7 ? "前 7 天" : "更早";
    groups.get(key)?.push(conversation);
  }

  return [...groups.entries()].filter(([, items]) => items.length > 0);
}

function startOfDay(value: Date) {
  return new Date(value.getFullYear(), value.getMonth(), value.getDate());
}

export async function copyText(text: string) {
  if (!text.trim()) return;
  await navigator.clipboard.writeText(text);
}

export function formatTime(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return new Intl.DateTimeFormat("zh-CN", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

export function shortArguments(value: string) {
  const compact = value.replace(/\s+/g, " ").trim();
  return compact.length > 72 ? `${compact.slice(0, 72)}…` : compact;
}
