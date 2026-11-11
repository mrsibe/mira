import { describe, expect, it } from "vitest";
import type { Conversation, Project } from "../mygo";
import { buildSidebarModel } from "./sidebar";

const projects: Project[] = [
  { id: "later", name: "Later", createdAt: "2026-11-10T08:00:00Z" },
  { id: "earlier", name: "Earlier", createdAt: "2026-11-10T07:00:00Z" },
];

function conversation(id: string, title: string, projectId?: string): Conversation {
  return {
    id,
    title,
    preview: "",
    projectId,
    createdAt: "2026-11-10T08:00:00Z",
    updatedAt: new Date().toISOString(),
    messageCount: 1,
  };
}

describe("buildSidebarModel", () => {
  it("sorts projects by creation time and keeps unfiled chats separate", () => {
    const model = buildSidebarModel(
      projects,
      [
        conversation("a", "Inside earlier", "earlier"),
        conversation("b", "Inside later", "later"),
        conversation("c", "No project"),
        conversation("d", "Unknown project", "missing"),
      ],
      "",
    );

    expect(model.projects.map((group) => group.project.id)).toEqual(["earlier", "later"]);
    expect(model.projects[0]?.items.map((item) => item.id)).toEqual(["a"]);
    expect(model.unfiled.flatMap(([, items]) => items.map((item) => item.id))).toEqual(
      expect.arrayContaining(["c", "d"]),
    );
  });

  it("filters by title and preview while retaining project groups", () => {
    const inProject = conversation("a", "Design notes", "earlier");
    inProject.preview = "Spacing review";
    const model = buildSidebarModel(
      projects,
      [inProject, conversation("b", "Other", "later")],
      "spacing",
    );

    expect(model.searching).toBe(true);
    expect(model.projects[0]?.items.map((item) => item.id)).toEqual(["a"]);
    expect(model.projects[1]?.items).toEqual([]);
  });
});
