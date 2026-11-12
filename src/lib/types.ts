import type { Spec } from "@json-render/core";
import type { UsageView } from "../mygo";

export type ToolRun = {
  id: string;
  name: string;
  arguments: string;
  result?: string;
  isError?: boolean;
  status: "running" | "done";
};

export type PlanStepStatus = "pending" | "in_progress" | "completed";

export type PlanStep = {
  title: string;
  status: PlanStepStatus;
};

export type PlanView = {
  steps: PlanStep[];
};

export type RunPart =
  | { id: string; kind: "thinking"; text: string; active: boolean }
  | { id: string; kind: "text"; text: string }
  | { id: string; kind: "tool"; tool: ToolRun };

export type RunView = {
  status: "running" | "error";
  parts: RunPart[];
  nextPartId: number;
  startedAt: number;
  thinkingEndedAt?: number;
  usage?: UsageView;
  error?: string;
};

export type UISpecStatus = "streaming" | "complete" | "invalid";

export type UISpecView = {
  spec: Spec;
  status: UISpecStatus;
  error?: string;
};

export type AssistantSegment =
  | { kind: "text"; key: string; text: string }
  | { kind: "spec"; key: string; spec: UISpecView };

export type ThemeMode = "system" | "light" | "dark";

export type SettingsSection =
  | "general"
  | "personalization"
  | "memory"
  | "search"
  | "model"
  | "about";

export type SearchProvider = "duckduckgo" | "tavily";

export type ReasoningEffort = "auto" | "low" | "medium" | "high";
