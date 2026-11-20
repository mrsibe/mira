import { parseAssistantAnswer } from "../genui/stream";
import type { Messages } from "../locales";
import type { ToolResultView, UIMessage, UsageView } from "../mygo";
import { shortArguments } from "./format";
import { activePlanStep, PLAN_TOOL_NAME, parsePlan } from "./plan";
import type { AssistantSegment, PlanStep, PlanView, RunView, ToolRun, UISpecView } from "./types";

export type ThreadItem =
  | { kind: "user"; key: string; message: UIMessage }
  | { kind: "assistant"; key: string; messages: UIMessage[] };

export type ActivityItem =
  | { kind: "thinking"; key: string; text: string }
  | { kind: "plan"; key: string; steps: PlanStep[] }
  | { kind: "tool"; key: string; tool: ToolRun };

export type Activity = {
  items: ActivityItem[];
  plan: PlanView | null;
};

export type AssistantTurnView = {
  activity: Activity;
  answer: string;
  spec: UISpecView | null;
  segments: AssistantSegment[];
  messageIndex?: number;
  usage?: UsageView;
  error?: string;
  streaming: boolean;
  thinkingDurationMs?: number;
};

export function groupMessages(messages: UIMessage[]): ThreadItem[] {
  const items: ThreadItem[] = [];
  for (const message of messages) {
    if (message.role === "user") {
      items.push({ kind: "user", key: `user-${message.index}`, message });
      continue;
    }
    if (message.role !== "assistant" && message.role !== "tool") continue;

    const previous = items[items.length - 1];
    if (previous?.kind === "assistant") {
      previous.messages.push(message);
    } else {
      items.push({
        kind: "assistant",
        key: `assistant-${message.index}`,
        messages: [message],
      });
    }
  }
  return items;
}

export function collectToolResults(messages: UIMessage[]) {
  const results = new Map<string, ToolResultView>();
  for (const message of messages) {
    if (message.role !== "tool" || !message.toolCallId) continue;
    results.set(message.toolCallId, {
      id: message.toolCallId,
      name: message.toolName ?? "tool",
      text: messageText(message),
      isError: message.isError,
    });
  }
  return results;
}

export function messageText(message: UIMessage) {
  return message.blocks
    .filter((block) => block.type === "text")
    .map((block) => block.text ?? "")
    .join("\n\n");
}

export function answerText(message: UIMessage) {
  let start = 0;
  for (const [index, block] of message.blocks.entries()) {
    if (block.type !== "text") start = index + 1;
  }
  return message.blocks
    .slice(start)
    .filter((block) => block.type === "text")
    .map((block) => block.text ?? "")
    .join("\n\n");
}

export function buildCompletedTurn(
  messages: UIMessage[],
  toolResults: Map<string, ToolResultView>,
  t: Messages,
): AssistantTurnView {
  let answerMessage: UIMessage | undefined;
  for (let index = messages.length - 1; index >= 0; index--) {
    const message = messages[index];
    if (message?.role !== "assistant") continue;
    if (message.blocks.some((block) => block.type === "tool_call" && block.toolCall)) continue;
    if (!answerText(message).trim()) continue;
    answerMessage = message;
    break;
  }

  const rawAnswer = answerMessage ? answerText(answerMessage) : "";
  const parsed = parseAssistantAnswer(rawAnswer, { messages: t });
  return {
    activity: completedActivity(messages, toolResults, t),
    answer: parsed.text,
    spec: parsed.spec,
    segments: parsed.segments,
    messageIndex: answerMessage?.index,
    usage: answerMessage?.usage,
    streaming: false,
    thinkingDurationMs: answerMessage?.thinkingDurationMs,
  };
}

function completedActivity(
  messages: UIMessage[],
  toolResults: Map<string, ToolResultView>,
  t: Messages,
): Activity {
  const items: ActivityItem[] = [];
  const seenCalls = new Set<string>();
  let plan: PlanView | null = null;

  for (const [messageIndex, message] of messages.entries()) {
    if (message.role === "tool") {
      if (message.toolName === PLAN_TOOL_NAME) continue;
      if (message.toolCallId && seenCalls.has(message.toolCallId)) continue;
      items.push({
        kind: "tool",
        key: `result-${messageIndex}`,
        tool: {
          id: message.toolCallId ?? message.toolName ?? `tool-${messageIndex}`,
          name: message.toolName ?? "tool",
          arguments: "",
          result: messageText(message),
          isError: message.isError,
          status: "done",
        },
      });
      continue;
    }
    if (message.role !== "assistant") continue;

    for (const [blockIndex, block] of message.blocks.entries()) {
      if (block.type === "thinking" && block.text?.trim()) {
        items.push({
          kind: "thinking",
          key: `thinking-${messageIndex}-${blockIndex}`,
          text: block.text.trim(),
        });
        continue;
      }
      if (block.type !== "tool_call" || !block.toolCall) continue;
      const call = block.toolCall;
      seenCalls.add(call.id);
      if (call.name === PLAN_TOOL_NAME) {
        const nextPlan = parsePlan(call.arguments);
        if (!nextPlan) continue;
        const previousPlan = plan;
        plan = nextPlan;
        const steps = planDelta(previousPlan, nextPlan);
        if (steps.length > 0) {
          items.push({
            kind: "plan",
            key: `plan-${messageIndex}-${blockIndex}`,
            steps,
          });
        }
        continue;
      }
      const result = toolResults.get(call.id);
      items.push({
        kind: "tool",
        key: `tool-${call.id}`,
        tool: {
          id: call.id,
          name: call.name,
          arguments: call.arguments,
          result: result?.text ?? t.toolResultIncomplete(),
          isError: result?.isError ?? true,
          status: "done",
        },
      });
    }
  }
  return { items, plan };
}

function planDelta(previous: PlanView | null, next: PlanView) {
  if (!previous || previous.steps.length !== next.steps.length) return next.steps;
  const changed: PlanStep[] = [];
  for (const [index, step] of next.steps.entries()) {
    const before = previous.steps[index];
    if (!before || before.title !== step.title || before.status !== step.status) {
      changed.push(step);
    }
  }
  return changed;
}

export function buildLiveTurn(run: RunView, t: Messages): AssistantTurnView {
  let plan: PlanView | null = null;
  const items: ActivityItem[] = [];
  let boundary = -1;

  for (const [index, part] of run.parts.entries()) {
    if (part.kind === "thinking") {
      boundary = index;
      if (part.text.trim()) {
        items.push({ kind: "thinking", key: part.id, text: part.text.trim() });
      }
      continue;
    }
    if (part.kind !== "tool") continue;
    boundary = index;
    if (part.tool.name === PLAN_TOOL_NAME) {
      const nextPlan = parsePlan(part.tool.arguments);
      if (!nextPlan) continue;
      const previousPlan = plan;
      plan = nextPlan;
      const steps = planDelta(previousPlan, nextPlan);
      if (steps.length > 0) {
        items.push({ kind: "plan", key: part.id, steps });
      }
      continue;
    }
    items.push({ kind: "tool", key: part.id, tool: part.tool });
  }

  const rawAnswer = run.parts
    .slice(boundary + 1)
    .filter((part) => part.kind === "text")
    .map((part) => (part.kind === "text" ? part.text : ""))
    .join("");
  const parsed = parseAssistantAnswer(rawAnswer, {
    streaming: run.status === "running",
    messages: t,
  });

  return {
    activity: { items, plan },
    answer: parsed.text,
    spec: parsed.spec,
    segments: parsed.segments,
    usage: run.usage,
    error: run.error,
    streaming: run.status === "running",
    thinkingDurationMs:
      run.thinkingEndedAt === undefined
        ? undefined
        : Math.max(0, run.thinkingEndedAt - run.startedAt),
  };
}

export function sumUsage(messages: UIMessage[], extra?: UsageView): UsageView {
  const total: UsageView = {
    promptTokens: 0,
    completionTokens: 0,
    cacheReadTokens: 0,
    cacheWriteTokens: 0,
    reasoningTokens: 0,
    totalTokens: 0,
  };
  for (const message of messages) {
    if (message.usage) addUsage(total, message.usage);
  }
  if (extra) addUsage(total, extra);
  return total;
}

function addUsage(total: UsageView, usage: UsageView) {
  total.promptTokens += usage.promptTokens;
  total.completionTokens += usage.completionTokens;
  total.cacheReadTokens = (total.cacheReadTokens ?? 0) + (usage.cacheReadTokens ?? 0);
  total.cacheWriteTokens = (total.cacheWriteTokens ?? 0) + (usage.cacheWriteTokens ?? 0);
  total.reasoningTokens = (total.reasoningTokens ?? 0) + (usage.reasoningTokens ?? 0);
  total.totalTokens += usage.totalTokens;
}

export function liveActivityTitle(activity: Activity, streaming: boolean, t: Messages) {
  if (!streaming) return t.activityStopped();
  let runningTool: ToolRun | undefined;
  for (let index = activity.items.length - 1; index >= 0; index--) {
    const item = activity.items[index];
    if (item?.kind === "tool" && item.tool.status === "running") {
      runningTool = item.tool;
      break;
    }
  }
  if (runningTool) return toolActivityLabel(runningTool, t);
  const active = activePlanStep(activity.plan);
  if (active) return activeTitle(active.title, t);
  if (activity.plan?.steps.some((step) => step.status === "pending")) {
    return t.activityContinuingPlan();
  }
  if (activity.items.length > 0) return t.activityOrganizing();
  return t.activityThinking();
}

export function activeTitle(title: string, t: Messages) {
  return t.activityActiveTitle(title);
}

export function toolActivityLabel(tool: ToolRun, t: Messages) {
  if (tool.name === "web_search") {
    const query = argumentString(tool.arguments, "query");
    if (tool.status === "running") {
      return query ? t.activitySearchingQuery(query) : t.activitySearchingWeb();
    }
    if (tool.isError) {
      return query ? t.activitySearchFailedQuery(query) : t.activitySearchFailed();
    }
    return query ? t.activitySearchedQuery(query) : t.activitySearchedWeb();
  }
  if (tool.name === "search_memory") {
    const query = argumentString(tool.arguments, "query");
    if (tool.status === "running") {
      return query ? t.activitySearchingMemoryQuery(query) : t.activitySearchingMemory();
    }
    if (tool.isError) {
      return query ? t.activityMemoryFailedQuery(query) : t.activityMemoryFailed();
    }
    return query ? t.activitySearchedMemoryQuery(query) : t.activitySearchedMemory();
  }
  if (tool.name === "get_current_time") {
    return tool.status === "running" ? t.activityGettingTime() : t.activityGotTime();
  }
  const preview = toolPreview(tool, t);
  const label = toolLabel(tool.name, t);
  if (tool.status === "running") {
    return preview
      ? t.activityRunningToolWithPreview(label, preview)
      : t.activityRunningTool(label);
  }
  return preview ? t.activityRanToolWithPreview(label, preview) : t.activityRanTool(label);
}

function toolLabel(name: string, t: Messages) {
  if (name === "web_search") return t.toolLabelWebSearch();
  if (name === "search_memory") return t.toolLabelSearchMemory();
  if (name === "get_current_time") return t.toolLabelCurrentTime();
  return name;
}

function toolPreview(tool: ToolRun, t: Messages) {
  if (tool.name === "web_search" || tool.name === "search_memory") {
    const query = argumentString(tool.arguments, "query");
    if (query) return t.toolPreviewQuery(query);
  }
  return tool.arguments ? shortArguments(tool.arguments) : "";
}

function argumentString(argumentsText: string, key: string) {
  try {
    const value: unknown = JSON.parse(argumentsText);
    if (!value || typeof value !== "object" || !(key in value)) return "";
    const field = (value as Record<string, unknown>)[key];
    return typeof field === "string" ? field.trim() : "";
  } catch {
    return "";
  }
}

export function formatThoughtDuration(milliseconds: number) {
  const totalSeconds = Math.max(1, Math.round(milliseconds / 1000));
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  if (hours > 0) {
    return `${hours}h${minutes > 0 ? `${minutes}m` : ""}${seconds > 0 ? `${seconds}s` : ""}`;
  }
  if (minutes > 0) return `${minutes}m${seconds > 0 ? `${seconds}s` : ""}`;
  return `${seconds}s`;
}
