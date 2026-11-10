import { Check } from "lucide-react";
import { activeTitle } from "../../lib/activity";
import type { PlanStep } from "../../lib/types";

export function PlanList({ steps }: { steps: PlanStep[] }) {
  return (
    <div className="plan">
      <ol className="plan-list" aria-label="执行计划">
        {steps.map((step) => (
          <li
            key={step.title}
            className={`plan-step ${step.status}`}
            aria-current={step.status === "in_progress" ? "step" : undefined}
          >
            <span className="plan-marker" aria-hidden="true">
              {step.status === "completed" ? (
                <Check size={12} />
              ) : (
                <span className={`plan-dot ${step.status}`} />
              )}
            </span>
            <span className="plan-title">
              {step.status === "in_progress" ? activeTitle(step.title) : step.title}
            </span>
          </li>
        ))}
      </ol>
    </div>
  );
}
