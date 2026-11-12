import {
  ChevronRight,
  Folder,
  FolderInput,
  MoreHorizontal,
  Plus,
  Search,
  Settings,
  SquarePen,
} from "lucide-react";
import {
  type DragEvent as ReactDragEvent,
  type KeyboardEvent as ReactKeyboardEvent,
  type MouseEvent as ReactMouseEvent,
  type PointerEvent as ReactPointerEvent,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";
import { buildSidebarModel } from "../lib/sidebar";
import {
  type ConversationDrag,
  canDropConversation,
  conversationDragMime,
  conversationProjectID,
} from "../lib/sidebarDrag";
import {
  applySidebarWidth,
  clampSidebarWidthForViewport,
  defaultSidebarWidth,
  loadPreferredSidebarWidth,
  maxSidebarWidthForViewport,
  minSidebarWidth,
  savePreferredSidebarWidth,
} from "../lib/sidebarWidth";
import type { Conversation, Project } from "../mygo";
import { IconButton } from "../ui";
import { ConversationMenu, ProjectMenu } from "./SidebarMenus";

type SidebarProps = {
  conversations: Conversation[];
  projects: Project[];
  activeId: string | null;
  onOpenSearch: () => void;
  onSelect: (id: string) => void;
  onNew: () => void;
  onNewInProject: (projectID: string) => void;
  onRename: (id: string, title: string) => void;
  onDelete: (id: string) => void;
  onMove: (id: string, projectID: string) => Promise<boolean>;
  onCreateProject: (afterMoveConversationID?: string) => void;
  onRenameProject: (project: Project) => void;
  onDeleteProject: (project: Project) => void;
  onSettings: () => void;
};

type MenuState =
  | { kind: "conversation"; conversation: Conversation; anchor: HTMLElement }
  | { kind: "project"; project: Project; anchor: HTMLElement }
  | null;

export function Sidebar({
  conversations,
  projects,
  activeId,
  onOpenSearch,
  onSelect,
  onNew,
  onNewInProject,
  onRename,
  onDelete,
  onMove,
  onCreateProject,
  onRenameProject,
  onDeleteProject,
  onSettings,
}: SidebarProps) {
  const [editingId, setEditingId] = useState<string | null>(null);
  const [draft, setDraft] = useState("");
  const [expandedProjects, setExpandedProjects] = useState<Set<string>>(() => new Set());
  const [menu, setMenu] = useState<MenuState>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const [viewportWidth, setViewportWidth] = useState(() => window.innerWidth);
  const [preferredSidebarWidth, setPreferredSidebarWidth] = useState(loadPreferredSidebarWidth);
  const [resizing, setResizing] = useState(false);
  const [dragging, setDragging] = useState<ConversationDrag | null>(null);
  const [dropTarget, setDropTarget] = useState<string | null>(null);
  const sidebarWidth = clampSidebarWidthForViewport(preferredSidebarWidth, viewportWidth);
  const sidebarWidthRef = useRef(sidebarWidth);
  const dragRef = useRef<{ startX: number; startWidth: number } | null>(null);
  const conversationDragRef = useRef<ConversationDrag | null>(null);
  const dropTargetRef = useRef<string | null>(null);
  const expandTimerRef = useRef<number | undefined>(undefined);
  const listRef = useRef<HTMLElement>(null);
  const scrollFrameRef = useRef<number | null>(null);
  const scrollSpeedRef = useRef(0);
  sidebarWidthRef.current = sidebarWidth;

  useEffect(() => {
    if (editingId) inputRef.current?.focus();
  }, [editingId]);

  useEffect(() => {
    return () => {
      if (expandTimerRef.current !== undefined) {
        window.clearTimeout(expandTimerRef.current);
      }
      if (scrollFrameRef.current !== null && typeof cancelAnimationFrame === "function") {
        cancelAnimationFrame(scrollFrameRef.current);
      }
      document.body.classList.remove("conversation-dragging");
    };
  }, []);

  useEffect(() => {
    applySidebarWidth(sidebarWidth);
  }, [sidebarWidth]);

  useEffect(() => {
    function onViewportResize() {
      setViewportWidth(window.innerWidth);
    }
    window.addEventListener("resize", onViewportResize);
    return () => window.removeEventListener("resize", onViewportResize);
  }, []);

  useEffect(() => {
    if (!resizing) return;

    function onPointerMove(event: PointerEvent) {
      const drag = dragRef.current;
      if (!drag) return;
      const next = clampSidebarWidthForViewport(
        drag.startWidth + event.clientX - drag.startX,
        viewportWidth,
      );
      sidebarWidthRef.current = next;
      setPreferredSidebarWidth(next);
    }

    function finishResize() {
      setResizing(false);
      dragRef.current = null;
      document.body.classList.remove("sidebar-resizing");
      savePreferredSidebarWidth(sidebarWidthRef.current);
    }

    window.addEventListener("pointermove", onPointerMove);
    window.addEventListener("pointerup", finishResize);
    window.addEventListener("pointercancel", finishResize);
    window.addEventListener("blur", finishResize);
    return () => {
      window.removeEventListener("pointermove", onPointerMove);
      window.removeEventListener("pointerup", finishResize);
      window.removeEventListener("pointercancel", finishResize);
      window.removeEventListener("blur", finishResize);
    };
  }, [resizing, viewportWidth]);

  const closeMenu = useCallback(() => {
    setMenu(null);
  }, []);

  const model = buildSidebarModel(projects, conversations);
  const canUnfile = dragging !== null && dragging.sourceProjectID !== "";
  const showUnfiledRegion = model.unfiled.length > 0 || canUnfile;

  function startRename(item: Conversation) {
    setEditingId(item.id);
    setDraft(item.title);
  }

  function commitRename() {
    if (!editingId) return;
    const title = draft.trim();
    if (title) onRename(editingId, title);
    setEditingId(null);
  }

  function toggleProject(id: string) {
    setExpandedProjects((current) => {
      const next = new Set(current);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  function expandProject(id: string) {
    setExpandedProjects((current) => {
      if (current.has(id)) return current;
      const next = new Set(current);
      next.add(id);
      return next;
    });
  }

  function updateDropTarget(projectID: string | null) {
    dropTargetRef.current = projectID;
    setDropTarget(projectID);
  }

  function cancelExpand() {
    if (expandTimerRef.current === undefined) return;
    window.clearTimeout(expandTimerRef.current);
    expandTimerRef.current = undefined;
  }

  function scheduleExpand(projectID: string) {
    if (expandedProjects.has(projectID) || expandTimerRef.current !== undefined) return;
    expandTimerRef.current = window.setTimeout(() => {
      expandTimerRef.current = undefined;
      expandProject(projectID);
    }, 650);
  }

  function stopAutoScroll() {
    scrollSpeedRef.current = 0;
    if (scrollFrameRef.current !== null && typeof cancelAnimationFrame === "function") {
      cancelAnimationFrame(scrollFrameRef.current);
    }
    scrollFrameRef.current = null;
  }

  function tickAutoScroll() {
    const list = listRef.current;
    if (!list || scrollSpeedRef.current === 0) {
      scrollFrameRef.current = null;
      return;
    }
    const before = list.scrollTop;
    list.scrollTop = before + scrollSpeedRef.current;
    if (list.scrollTop === before) {
      scrollFrameRef.current = null;
      return;
    }
    scrollFrameRef.current = requestAnimationFrame(tickAutoScroll);
  }

  function updateAutoScroll(event: ReactDragEvent<HTMLElement>) {
    const list = listRef.current;
    if (!list) return;
    const rect = list.getBoundingClientRect();
    if (rect.height <= 0) return;
    const threshold = Math.min(48, rect.height / 4);
    let speed = 0;
    if (event.clientY < rect.top + threshold) speed = -8;
    else if (event.clientY > rect.bottom - threshold) speed = 8;
    scrollSpeedRef.current = speed;
    if (speed === 0) {
      stopAutoScroll();
      return;
    }
    if (scrollFrameRef.current === null && typeof requestAnimationFrame === "function") {
      scrollFrameRef.current = requestAnimationFrame(tickAutoScroll);
    }
  }

  function finishConversationDrag() {
    conversationDragRef.current = null;
    setDragging(null);
    updateDropTarget(null);
    cancelExpand();
    stopAutoScroll();
    document.body.classList.remove("conversation-dragging");
  }

  function beginConversationDrag(
    event: ReactDragEvent<HTMLButtonElement>,
    conversation: Conversation,
  ) {
    const drag: ConversationDrag = {
      conversationID: conversation.id,
      sourceProjectID: conversationProjectID(conversation),
    };
    event.dataTransfer.effectAllowed = "move";
    event.dataTransfer.setData(conversationDragMime, conversation.id);
    event.dataTransfer.setData("text/plain", conversation.title);
    conversationDragRef.current = drag;
    setDragging(drag);
    updateDropTarget(null);
    setMenu(null);
    document.body.classList.add("conversation-dragging");
  }

  function dragOverProject(event: ReactDragEvent<HTMLElement>, projectID: string) {
    if (!canDropConversation(conversationDragRef.current, projectID)) return;
    event.preventDefault();
    event.dataTransfer.dropEffect = "move";
    if (dropTargetRef.current !== projectID) {
      cancelExpand();
      updateDropTarget(projectID);
    }
    scheduleExpand(projectID);
    updateAutoScroll(event);
  }

  function dragLeaveProject(event: ReactDragEvent<HTMLElement>, projectID: string) {
    const related = event.relatedTarget;
    if (related instanceof Node && event.currentTarget.contains(related)) return;
    if (dropTargetRef.current === projectID) updateDropTarget(null);
    cancelExpand();
    stopAutoScroll();
  }

  function dropOnProject(event: ReactDragEvent<HTMLElement>, projectID: string) {
    event.preventDefault();
    event.stopPropagation();
    const drag = conversationDragRef.current;
    if (!drag || !canDropConversation(drag, projectID)) {
      finishConversationDrag();
      return;
    }
    finishConversationDrag();
    void onMove(drag.conversationID, projectID).then((moved) => {
      if (moved) expandProject(projectID);
    });
  }

  function dragOverUnfiled(event: ReactDragEvent<HTMLElement>) {
    if (!canDropConversation(conversationDragRef.current, "")) return;
    event.preventDefault();
    event.dataTransfer.dropEffect = "move";
    cancelExpand();
    if (dropTargetRef.current !== "") updateDropTarget("");
    updateAutoScroll(event);
  }

  function dragLeaveUnfiled(event: ReactDragEvent<HTMLElement>) {
    const related = event.relatedTarget;
    if (related instanceof Node && event.currentTarget.contains(related)) return;
    if (dropTargetRef.current === "") updateDropTarget(null);
    stopAutoScroll();
  }

  function dropOnUnfiled(event: ReactDragEvent<HTMLElement>) {
    event.preventDefault();
    const drag = conversationDragRef.current;
    if (!drag || !canDropConversation(drag, "")) {
      finishConversationDrag();
      return;
    }
    finishConversationDrag();
    void onMove(drag.conversationID, "");
  }

  function openConversationMenu(event: ReactMouseEvent<HTMLButtonElement>, item: Conversation) {
    event.stopPropagation();
    setMenu({ kind: "conversation", conversation: item, anchor: event.currentTarget });
  }

  function openProjectMenu(event: ReactMouseEvent<HTMLButtonElement>, project: Project) {
    event.stopPropagation();
    setMenu({ kind: "project", project, anchor: event.currentTarget });
  }

  function startResize(event: ReactPointerEvent<HTMLHRElement>) {
    if (event.button !== 0) return;
    event.preventDefault();
    dragRef.current = { startX: event.clientX, startWidth: sidebarWidth };
    setResizing(true);
    document.body.classList.add("sidebar-resizing");
  }

  function updateSidebarWidth(value: number) {
    const next = clampSidebarWidthForViewport(value, viewportWidth);
    setPreferredSidebarWidth(next);
    savePreferredSidebarWidth(next);
  }

  function resetSidebarWidth() {
    updateSidebarWidth(defaultSidebarWidth);
  }

  function onResizerKeyDown(event: ReactKeyboardEvent<HTMLHRElement>) {
    const step = event.shiftKey ? 48 : 16;
    if (event.key === "ArrowLeft") {
      event.preventDefault();
      updateSidebarWidth(sidebarWidth - step);
    } else if (event.key === "ArrowRight") {
      event.preventDefault();
      updateSidebarWidth(sidebarWidth + step);
    } else if (event.key === "Home") {
      event.preventDefault();
      updateSidebarWidth(minSidebarWidth);
    } else if (event.key === "End") {
      event.preventDefault();
      updateSidebarWidth(maxSidebarWidthForViewport(viewportWidth));
    }
  }

  function renderConversation(item: Conversation) {
    const active = item.id === activeId;
    const editing = item.id === editingId;
    const menuOpen = menu?.kind === "conversation" && menu.conversation.id === item.id;
    return (
      <div
        key={item.id}
        className={`conversation-item ${active ? "active" : ""} ${menuOpen ? "menu-open" : ""} ${
          dragging?.conversationID === item.id ? "dragging" : ""
        }`}
      >
        {editing ? (
          <input
            ref={inputRef}
            className="rename-input"
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
            onBlur={commitRename}
            onKeyDown={(event) => {
              if (event.key === "Enter") commitRename();
              if (event.key === "Escape") setEditingId(null);
            }}
          />
        ) : (
          <button
            type="button"
            className="conversation-select"
            draggable
            aria-current={active ? "page" : undefined}
            onClick={() => onSelect(item.id)}
            onDoubleClick={() => startRename(item)}
            onDragStart={(event) => beginConversationDrag(event, item)}
            onDragEnd={finishConversationDrag}
            title={item.title}
          >
            <span>{item.title}</span>
          </button>
        )}
        {!editing && (
          <span className="conversation-actions">
            <button
              type="button"
              className="mini-button"
              title="对话操作"
              aria-label={`打开“${item.title}”的操作菜单`}
              aria-haspopup="menu"
              onClick={(event) => openConversationMenu(event, item)}
            >
              <MoreHorizontal size={16} />
            </button>
          </span>
        )}
      </div>
    );
  }

  return (
    <aside className={`sidebar ${resizing ? "resizing" : ""}`}>
      <div className="sidebar-actions">
        <button className="new-chat" type="button" onClick={onNew}>
          <SquarePen size={17} />
          <span>新对话</span>
        </button>
        <IconButton label="搜索对话" className="sidebar-search-button" onClick={onOpenSearch}>
          <Search size={17} />
        </IconButton>
      </div>

      <nav ref={listRef} className="conversation-list" aria-label="对话与项目列表">
        <div className="sidebar-section-head">
          <h2>项目</h2>
          <IconButton
            label="新建项目"
            className="sidebar-section-add"
            onClick={() => onCreateProject()}
          >
            <Plus size={15} />
          </IconButton>
        </div>

        {model.projects.map(({ project, items }) => {
          const expanded = expandedProjects.has(project.id);
          const hasActive = items.some((item) => item.id === activeId);
          const menuOpen = menu?.kind === "project" && menu.project.id === project.id;
          return (
            // biome-ignore lint/a11y/noStaticElementInteractions: drag targets are pointer-only; project actions remain available through buttons and menus
            <section
              key={project.id}
              className={`project-group ${dropTarget === project.id ? "drop-target" : ""}`}
              onDragOver={(event) => dragOverProject(event, project.id)}
              onDragLeave={(event) => dragLeaveProject(event, project.id)}
              onDrop={(event) => dropOnProject(event, project.id)}
            >
              <div
                className={`project-row ${hasActive ? "has-active" : ""} ${menuOpen ? "menu-open" : ""}`}
              >
                <button
                  type="button"
                  className="project-toggle"
                  aria-expanded={expanded}
                  onClick={() => toggleProject(project.id)}
                >
                  <ChevronRight
                    size={14}
                    className={`project-chevron ${expanded ? "expanded" : ""}`}
                  />
                  <Folder size={15} />
                  <span className="project-name">{project.name}</span>
                </button>
                <span className="project-actions">
                  <button
                    type="button"
                    className="mini-button"
                    title="在此项目中新建对话"
                    aria-label={`在“${project.name}”中新建对话`}
                    onClick={(event) => {
                      event.stopPropagation();
                      onNewInProject(project.id);
                    }}
                  >
                    <Plus size={15} />
                  </button>
                  <button
                    type="button"
                    className="mini-button"
                    title="项目操作"
                    aria-label={`打开“${project.name}”的项目菜单`}
                    aria-haspopup="menu"
                    onClick={(event) => openProjectMenu(event, project)}
                  >
                    <MoreHorizontal size={16} />
                  </button>
                </span>
              </div>
              {expanded && (
                <div className="project-conversations">
                  {items.length === 0 ? (
                    <p className="project-empty">还没有对话</p>
                  ) : (
                    items.map(renderConversation)
                  )}
                </div>
              )}
            </section>
          );
        })}

        {model.projects.length === 0 && conversations.length > 0 && (
          <p className="sidebar-empty compact">还没有项目</p>
        )}

        {showUnfiledRegion && (
          // biome-ignore lint/a11y/noStaticElementInteractions: the unfiled region becomes a pointer-only drop target during drag; the move menu remains the accessible alternative
          <div
            className={`unfiled-region ${canUnfile ? "drag-ready" : ""} ${
              dropTarget === "" ? "drop-target" : ""
            }`}
            onDragOver={dragOverUnfiled}
            onDragLeave={dragLeaveUnfiled}
            onDrop={dropOnUnfiled}
          >
            {canUnfile && (
              <div className="unfiled-drop-label">
                <FolderInput size={15} aria-hidden="true" />
                <span>拖到这里移出项目</span>
              </div>
            )}
            {model.unfiled.map(([label, items]) => (
              <section key={label} className="conversation-group">
                <h2>{label}</h2>
                {items.map(renderConversation)}
              </section>
            ))}
          </div>
        )}

        {conversations.length === 0 && projects.length === 0 && (
          <p className="sidebar-empty">还没有对话</p>
        )}
      </nav>

      <div className="sidebar-foot">
        <button type="button" className="foot-row" onClick={onSettings}>
          <Settings size={16} />
          <span>设置</span>
        </button>
      </div>

      <hr
        className={`sidebar-resizer ${resizing ? "resizing" : ""}`}
        aria-orientation="vertical"
        aria-label="调整侧栏宽度"
        aria-valuemin={minSidebarWidth}
        aria-valuemax={maxSidebarWidthForViewport(viewportWidth)}
        aria-valuenow={sidebarWidth}
        aria-valuetext={`${sidebarWidth}px`}
        title="拖动调整侧栏宽度；双击恢复默认"
        tabIndex={0}
        onPointerDown={startResize}
        onDoubleClick={resetSidebarWidth}
        onKeyDown={onResizerKeyDown}
      />

      {menu?.kind === "conversation" && (
        <ConversationMenu
          conversation={menu.conversation}
          projects={projects}
          anchor={menu.anchor}
          onClose={closeMenu}
          onRename={() => startRename(menu.conversation)}
          onMove={async (projectID) => {
            const moved = await onMove(menu.conversation.id, projectID);
            if (moved && projectID) expandProject(projectID);
            return moved;
          }}
          onDelete={() => onDelete(menu.conversation.id)}
          onCreateProject={() => onCreateProject(menu.conversation.id)}
        />
      )}
      {menu?.kind === "project" && (
        <ProjectMenu
          project={menu.project}
          anchor={menu.anchor}
          onClose={closeMenu}
          onRename={() => onRenameProject(menu.project)}
          onDelete={() => onDeleteProject(menu.project)}
        />
      )}
    </aside>
  );
}
