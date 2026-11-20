import type { Messages } from "../locales";
import type { Conversation, UsageView } from "../mygo";

const day = 24 * 60 * 60 * 1000;

export type ConversationGroup = "today" | "yesterday" | "last7Days" | "earlier";

export function groupConversations(
  conversations: Conversation[],
): Array<[group: ConversationGroup, items: Conversation[]]> {
  const now = new Date();
  const today = startOfDay(now);
  const groups = new Map<ConversationGroup, Conversation[]>([
    ["today", []],
    ["yesterday", []],
    ["last7Days", []],
    ["earlier", []],
  ]);

  for (const conversation of conversations) {
    const updated = new Date(conversation.updatedAt);
    const age = today.getTime() - startOfDay(updated).getTime();
    const key: ConversationGroup =
      age <= 0 ? "today" : age <= day ? "yesterday" : age <= day * 7 ? "last7Days" : "earlier";
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

export function shortArguments(value: string) {
  const compact = value.replace(/\s+/g, " ").trim();
  return compact.length > 72 ? `${compact.slice(0, 72)}…` : compact;
}

export function formatTokens(value: number) {
  if (value < 1000) return String(value);
  if (value < 1_000_000) {
    return `${(value / 1000).toFixed(value < 10_000 ? 1 : 0).replace(/\.0$/, "")}k`;
  }
  return `${(value / 1_000_000).toFixed(1).replace(/\.0$/, "")}M`;
}

export function usageDetails(usage: UsageView, messages: Messages) {
  const parts = [
    messages.usageInput(usage.promptTokens),
    messages.usageOutput(usage.completionTokens),
  ];
  if (usage.cacheReadTokens) parts.push(messages.usageCacheRead(usage.cacheReadTokens));
  if (usage.cacheWriteTokens) parts.push(messages.usageCacheWrite(usage.cacheWriteTokens));
  if (usage.reasoningTokens) parts.push(messages.usageReasoning(usage.reasoningTokens));
  parts.push(messages.usageTotal(usage.totalTokens));
  return parts.join(" · ");
}
