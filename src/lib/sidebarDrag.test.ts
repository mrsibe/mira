import { describe, expect, it } from "vitest";
import { canDropConversation, conversationProjectID } from "./sidebarDrag";

describe("sidebar drag helpers", () => {
  it("treats a missing project id as unfiled", () => {
    expect(conversationProjectID({ projectId: undefined })).toBe("");
    expect(conversationProjectID({ projectId: "project" })).toBe("project");
  });

  it("rejects missing drags and drops into the source project", () => {
    expect(canDropConversation(null, "project")).toBe(false);
    expect(
      canDropConversation({ conversationID: "chat", sourceProjectID: "project" }, "project"),
    ).toBe(false);
  });

  it("accepts drops into another project or unfiled", () => {
    const drag = { conversationID: "chat", sourceProjectID: "project" };
    expect(canDropConversation(drag, "other")).toBe(true);
    expect(canDropConversation(drag, "")).toBe(true);
  });
});
