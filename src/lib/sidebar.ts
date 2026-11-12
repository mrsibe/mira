import type { Conversation, Project } from "../mygo";
import { groupConversations } from "./format";

export type ProjectGroup = {
  project: Project;
  items: Conversation[];
};

export type SidebarModel = {
  projects: ProjectGroup[];
  unfiled: Array<[label: string, items: Conversation[]]>;
};

export function buildSidebarModel(
  projects: Project[],
  conversations: Conversation[],
): SidebarModel {
  const orderedProjects = [...projects].sort(
    (left, right) => new Date(left.createdAt).getTime() - new Date(right.createdAt).getTime(),
  );
  const projectIDs = new Set(orderedProjects.map((project) => project.id));
  return {
    projects: orderedProjects.map((project) => ({
      project,
      items: conversations.filter((conversation) => conversation.projectId === project.id),
    })),
    unfiled: groupConversations(
      conversations.filter(
        (conversation) => !conversation.projectId || !projectIDs.has(conversation.projectId ?? ""),
      ),
    ),
  };
}
