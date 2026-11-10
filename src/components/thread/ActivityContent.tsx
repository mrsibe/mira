import { type Activity, liveActivityTitle } from "../../lib/activity";
import { PlanList } from "./PlanList";
import { ToolCard } from "./ToolCard";

export function ActivityContent({ activity }: { activity: Activity }) {
  return (
    <div className="activity-panel">
      {activity.items.map((item) => {
        if (item.kind === "thinking") {
          return (
            <p key={item.key} className="activity-thinking">
              {item.text}
            </p>
          );
        }
        if (item.kind === "plan") {
          return <PlanList key={item.key} steps={item.steps} />;
        }
        return <ToolCard key={item.key} tool={item.tool} />;
      })}
    </div>
  );
}

export function LiveActivity({ activity, streaming }: { activity: Activity; streaming: boolean }) {
  return (
    <div className="activity-live" aria-live="polite">
      {streaming && <span className="activity-pulse" />}
      <span>{liveActivityTitle(activity, streaming)}</span>
    </div>
  );
}
