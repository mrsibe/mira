import { messageText } from "../../lib/activity";
import type { UIMessage } from "../../mygo";

export function UserMessage({ message }: { message: UIMessage }) {
  return (
    <div className="message user">
      <div className="user-bubble">{messageText(message)}</div>
    </div>
  );
}
