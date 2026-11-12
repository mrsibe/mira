import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { Conversation, Project } from "../mygo";
import { Sidebar } from "./Sidebar";

const now = new Date("2026-01-01T00:00:00.000Z").toISOString();

const projects: Project[] = [
  { id: "work", name: "Work", createdAt: now },
  { id: "life", name: "Life", createdAt: now },
];

const conversations: Conversation[] = [
  conversation("project-chat", "Project chat", "work"),
  conversation("other-chat", "Other chat", "life"),
  conversation("loose-chat", "Loose chat"),
];

function conversation(id: string, title: string, projectId?: string): Conversation {
  return {
    id,
    title,
    preview: "",
    projectId,
    createdAt: now,
    updatedAt: now,
    messageCount: 1,
  };
}

function dataTransfer(): DataTransfer {
  const data = new Map<string, string>();
  return {
    effectAllowed: "all",
    dropEffect: "none",
    setData(type: string, value: string) {
      data.set(type, value);
    },
    getData(type: string) {
      return data.get(type) ?? "";
    },
  } as DataTransfer;
}

function projectSection(name: string): HTMLElement {
  const section = screen.getByText(name).closest("section");
  if (!(section instanceof HTMLElement)) throw new Error(`missing project section for ${name}`);
  return section;
}

function renderSidebar(
  options: {
    onMove?: (id: string, projectID: string) => Promise<boolean>;
    search?: string;
    conversations?: Conversation[];
  } = {},
) {
  const onMove = vi.fn(options.onMove ?? (async () => true));
  render(
    <Sidebar
      conversations={options.conversations ?? conversations}
      projects={projects}
      activeId={null}
      search={options.search ?? ""}
      onSearch={vi.fn()}
      onSelect={vi.fn()}
      onNew={vi.fn()}
      onNewInProject={vi.fn()}
      onRename={vi.fn()}
      onDelete={vi.fn()}
      onMove={onMove}
      onCreateProject={vi.fn()}
      onRenameProject={vi.fn()}
      onDeleteProject={vi.fn()}
      onSettings={vi.fn()}
    />,
  );
  return { onMove };
}

describe("Sidebar drag and drop", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("moves a conversation into another project", async () => {
    const { onMove } = renderSidebar();
    const transfer = dataTransfer();

    fireEvent.dragStart(screen.getByRole("button", { name: "Project chat" }), {
      dataTransfer: transfer,
    });
    fireEvent.dragOver(projectSection("Life"), { dataTransfer: transfer, clientY: 100 });
    expect(projectSection("Life").classList.contains("drop-target")).toBe(true);
    fireEvent.drop(projectSection("Life"), { dataTransfer: transfer });

    await waitFor(() => expect(onMove).toHaveBeenCalledWith("project-chat", "life"));
  });

  it("does not move a conversation into its current project", async () => {
    const { onMove } = renderSidebar();
    const transfer = dataTransfer();

    fireEvent.dragStart(screen.getByRole("button", { name: "Project chat" }), {
      dataTransfer: transfer,
    });
    fireEvent.dragOver(projectSection("Work"), { dataTransfer: transfer, clientY: 100 });
    fireEvent.drop(projectSection("Work"), { dataTransfer: transfer });

    await Promise.resolve();
    expect(onMove).not.toHaveBeenCalled();
  });

  it("shows a temporary target and moves a conversation out of its project", async () => {
    const { onMove } = renderSidebar();
    const transfer = dataTransfer();

    fireEvent.dragStart(screen.getByRole("button", { name: "Project chat" }), {
      dataTransfer: transfer,
    });
    const unfiled = screen.getByText("拖到这里移出项目").closest(".unfiled-region");
    if (!(unfiled instanceof HTMLElement)) throw new Error("missing unfiled drop target");
    fireEvent.dragOver(unfiled, { dataTransfer: transfer, clientY: 100 });
    expect(unfiled.classList.contains("drop-target")).toBe(true);
    fireEvent.drop(unfiled, { dataTransfer: transfer });

    await waitFor(() => expect(onMove).toHaveBeenCalledWith("project-chat", ""));
  });

  it("shows the unfiled drop region even when there are no unfiled chats", () => {
    renderSidebar({
      conversations: [
        conversation("project-chat", "Project chat", "work"),
        conversation("other-chat", "Other chat", "life"),
      ],
    });

    fireEvent.dragStart(screen.getByRole("button", { name: "Project chat" }), {
      dataTransfer: dataTransfer(),
    });

    const unfiled = screen.getByText("拖到这里移出项目").closest(".unfiled-region");
    expect(unfiled?.classList.contains("drag-ready")).toBe(true);
  });

  it("expands a collapsed project after hovering during a drag", () => {
    vi.useFakeTimers();
    renderSidebar();
    fireEvent.click(screen.getByRole("button", { name: "Life" }));
    expect(screen.queryByText("Other chat")).toBeNull();

    fireEvent.dragStart(screen.getByRole("button", { name: "Project chat" }), {
      dataTransfer: dataTransfer(),
    });
    fireEvent.dragOver(projectSection("Life"), { dataTransfer: dataTransfer(), clientY: 100 });
    act(() => vi.advanceTimersByTime(700));

    expect(screen.getByText("Other chat")).toBeTruthy();
  });

  it("disables dragging while searching", () => {
    renderSidebar({ search: "chat" });
    expect(screen.getByRole("button", { name: "Project chat" }).getAttribute("draggable")).toBe(
      "false",
    );
  });
});
