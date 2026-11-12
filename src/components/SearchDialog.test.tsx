import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeAll, describe, expect, it, vi } from "vitest";
import type { Conversation, Project, SearchResult } from "../mygo";
import { SearchDialog } from "./SearchDialog";

const now = "2026-01-01T00:00:00.000Z";

const projects: Project[] = [{ id: "work", name: "Work", createdAt: now }];

const designConversation = conversation(
  "design",
  "Design review",
  "Spacing and hierarchy notes",
  "work",
);
const tripConversation = conversation("trip", "Trip plan", "Weekend itinerary", undefined);
const conversations: Conversation[] = [designConversation, tripConversation];

function conversation(
  id: string,
  title: string,
  preview: string,
  projectId?: string,
): Conversation {
  return {
    id,
    title,
    preview,
    projectId,
    createdAt: now,
    updatedAt: now,
    messageCount: 1,
  };
}

function searchResult(
  item: Conversation,
  messageIndex: number,
  snippet: string,
  role?: string,
  matchCount?: number,
): SearchResult {
  return { conversation: item, messageIndex, snippet, role, matchCount };
}

beforeAll(() => {
  HTMLDialogElement.prototype.showModal = function showModal() {
    this.open = true;
  };
  HTMLDialogElement.prototype.close = function close() {
    this.open = false;
  };
});

describe("SearchDialog", () => {
  it("shows recent conversations when the query is empty", () => {
    render(
      <SearchDialog
        conversations={conversations}
        projects={projects}
        onSearch={vi.fn(async () => [])}
        onSelect={vi.fn()}
        onClose={vi.fn()}
      />,
    );

    expect(screen.getByText("最近对话")).toBeTruthy();
    expect(screen.getByText("Design review")).toBeTruthy();
    expect(screen.getByText("Trip plan")).toBeTruthy();
    expect(screen.getByText("Work")).toBeTruthy();
  });

  it("searches conversation contents through the backend", async () => {
    const onSearch = vi.fn(async (query: string) =>
      query === "itinerary"
        ? [searchResult(tripConversation, 1, "Weekend itinerary details", "assistant", 2)]
        : [searchResult(designConversation, 2, "Spacing review notes", "user")],
    );
    render(
      <SearchDialog
        conversations={conversations}
        projects={projects}
        onSearch={onSearch}
        onSelect={vi.fn()}
        onClose={vi.fn()}
      />,
    );
    const input = screen.getByRole("searchbox", { name: "搜索对话" });

    fireEvent.change(input, { target: { value: "itinerary" } });
    expect(await screen.findByText("Trip plan")).toBeTruthy();
    await waitFor(() => expect(onSearch).toHaveBeenCalledWith("itinerary"));

    fireEvent.change(input, { target: { value: "spacing" } });
    expect(await screen.findByText("Design review")).toBeTruthy();
    expect(screen.queryByText("Trip plan")).toBeNull();
  });

  it("selects a full-text result and closes the dialog", async () => {
    const onSelect = vi.fn();
    const onClose = vi.fn();
    render(
      <SearchDialog
        conversations={conversations}
        projects={projects}
        onSearch={async () => [searchResult(tripConversation, 3, "Weekend itinerary", "assistant")]}
        onSelect={onSelect}
        onClose={onClose}
      />,
    );

    fireEvent.change(screen.getByRole("searchbox", { name: "搜索对话" }), {
      target: { value: "itinerary" },
    });
    await screen.findByText("Trip plan");
    fireEvent.click(screen.getByRole("button", { name: "打开“Trip plan”" }));

    expect(onSelect).toHaveBeenCalledWith("trip", 3);
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("shows an empty state when nothing matches", async () => {
    render(
      <SearchDialog
        conversations={conversations}
        projects={projects}
        onSearch={async () => []}
        onSelect={vi.fn()}
        onClose={vi.fn()}
      />,
    );

    fireEvent.change(screen.getByRole("searchbox", { name: "搜索对话" }), {
      target: { value: "missing" },
    });

    expect(await screen.findByText("没有匹配的对话")).toBeTruthy();
  });
});
