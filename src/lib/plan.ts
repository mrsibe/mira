import type { PlanStep, PlanStepStatus, PlanView } from "./types";

export const PLAN_TOOL_NAME = "update_plan";

const statuses: ReadonlySet<PlanStepStatus> = new Set(["pending", "in_progress", "completed"]);

export function parsePlan(argumentsText: string): PlanView | null {
  let value: unknown;
  try {
    value = JSON.parse(argumentsText);
  } catch {
    return null;
  }
  if (!value || typeof value !== "object" || !("steps" in value)) return null;
  const rawSteps = (value as { steps?: unknown }).steps;
  if (!Array.isArray(rawSteps) || rawSteps.length < 1 || rawSteps.length > 8) {
    return null;
  }

  const steps: PlanStep[] = [];
  let inProgress = 0;
  for (const rawStep of rawSteps) {
    if (!rawStep || typeof rawStep !== "object") return null;
    const title = "title" in rawStep ? rawStep.title : undefined;
    const status = "status" in rawStep ? rawStep.status : undefined;
    if (typeof title !== "string" || typeof status !== "string") return null;
    const normalizedTitle = title.trim().replace(/\s+/g, " ");
    if (!normalizedTitle || [...normalizedTitle].length > 80) return null;
    if (!statuses.has(status as PlanStepStatus)) return null;
    if (status === "in_progress") inProgress++;
    steps.push({ title: normalizedTitle, status: status as PlanStepStatus });
  }
  if (inProgress > 1) return null;
  return { steps };
}

export function activePlanStep(plan: PlanView | null) {
  return plan?.steps.find((step) => step.status === "in_progress") ?? null;
}
