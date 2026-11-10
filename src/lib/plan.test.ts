import { describe, expect, it } from "vitest";
import { activePlanStep, parsePlan } from "./plan";

describe("parsePlan", () => {
  it("normalizes a valid plan and finds the active step", () => {
    const plan = parsePlan(
      JSON.stringify({
        steps: [
          { title: "  收集   资料 ", status: "completed" },
          { title: "分析问题", status: "in_progress" },
          { title: "给出结论", status: "pending" },
        ],
      }),
    );

    expect(plan?.steps[0]?.title).toBe("收集 资料");
    expect(activePlanStep(plan)?.title).toBe("分析问题");
  });

  it("rejects malformed and ambiguous plans", () => {
    expect(parsePlan("not json")).toBeNull();
    expect(parsePlan(JSON.stringify({ steps: [] }))).toBeNull();
    expect(
      parsePlan(
        JSON.stringify({
          steps: [
            { title: "一", status: "in_progress" },
            { title: "二", status: "in_progress" },
          ],
        }),
      ),
    ).toBeNull();
  });
});
