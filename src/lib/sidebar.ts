import type { Conversation, Project } from "../mygo";
import { groupConversations } from "./format";

export type ProjectGroup = {
  project: Project;
  items: Conversation[];
};

export type SidebarModel = {
  searching: boolean;
  projects: ProjectGroup[];
  unfiled: Array<[label: string, items: Conversation[]]>;
};

export function buildSidebarModel(
  projects: Project[],
  conversations: Conversation[],
  search: string,
): SidebarModel {
  const query = search.trim().toLocaleLowerCase();
  const filtered = query
    ? conversations.filter((conversation) => matchesSearch(conversation, query))
    : conversations;
  const orderedProjects = [...projects].sort(
    (left, right) => new Date(left.createdAt).getTime() - new Date(right.createdAt).getTime(),
  );
  const projectIDs = new Set(orderedProjects.map((project) => project.id));
  return {
    searching: query !== "",
    projects: orderedProjects.map((project) => ({
      project,
      items: filtered.filter((conversation) => conversation.projectId === project.id),
    })),
    unfiled: groupConversations(
      filtered.filter(
        (conversation) => !conversation.projectId || !projectIDs.has(conversation.projectId ?? ""),
      ),
    ),
  };
}

function matchesSearch(conversation: Conversation, query: string) {
  return (
    conversation.title.toLocaleLowerCase().includes(query) ||
    conversation.preview.toLocaleLowerCase().includes(query)
  );
}
