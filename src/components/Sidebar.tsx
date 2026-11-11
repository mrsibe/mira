import {
  ChevronRight,
  Folder,
  MoreHorizontal,
  Plus,
  Search,
  Settings,
  SquarePen,
  X,
} from "lucide-react";
import {
  type MouseEvent as ReactMouseEvent,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";
import { buildSidebarModel } from "../lib/sidebar";
import type { Conversation, Project } from "../mygo";
import { IconButton } from "../ui";
import { ConversationMenu, ProjectMenu } from "./SidebarMenus";

type SidebarProps = {
  conversations: Conversation[];
  projects: Project[];
  activeId: string | null;
  search: string;
  onSearch: (value: string) => void;
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
  search,
  onSearch,
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
  const [collapsed, setCollapsed] = useState<Set<string>>(() => new Set());
  const [menu, setMenu] = useState<MenuState>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (editingId) inputRef.current?.focus();
  }, [editingId]);

  const closeMenu = useCallback(() => {
    setMenu(null);
  }, []);

  const model = buildSidebarModel(projects, conversations, search);
  const visibleProjects = model.projects.filter(
    (group) => !model.searching || group.items.length > 0,
  );
  const unfiledCount = model.unfiled.reduce((total, [, items]) => total + items.length, 0);

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
    setCollapsed((current) => {
      const next = new Set(current);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  function expandProject(id: string) {
    setCollapsed((current) => {
      if (!current.has(id)) return current;
      const next = new Set(current);
      next.delete(id);
      return next;
    });
  }

  function openConversationMenu(event: ReactMouseEvent<HTMLButtonElement>, item: Conversation) {
    event.stopPropagation();
    setMenu({ kind: "conversation", conversation: item, anchor: event.currentTarget });
  }

  function openProjectMenu(event: ReactMouseEvent<HTMLButtonElement>, project: Project) {
    event.stopPropagation();
    setMenu({ kind: "project", project, anchor: event.currentTarget });
  }

  function renderConversation(item: Conversation) {
    const active = item.id === activeId;
    const editing = item.id === editingId;
    const menuOpen = menu?.kind === "conversation" && menu.conversation.id === item.id;
    return (
      <div
        key={item.id}
        className={`conversation-item ${active ? "active" : ""} ${menuOpen ? "menu-open" : ""}`}
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
            aria-current={active ? "page" : undefined}
            onClick={() => onSelect(item.id)}
            onDoubleClick={() => startRename(item)}
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
    <aside className="sidebar">
      <button className="new-chat" type="button" onClick={onNew}>
        <SquarePen size={17} />
        <span>新对话</span>
      </button>

      <label className="search-box">
        <Search size={15} aria-hidden="true" />
        <input
          value={search}
          onChange={(event) => onSearch(event.target.value)}
          placeholder="搜索对话"
          aria-label="搜索对话"
        />
        {search && (
          <button type="button" className="search-clear" onClick={() => onSearch("")} title="清除">
            <X size={13} />
          </button>
        )}
      </label>

      <nav className="conversation-list" aria-label="对话与项目列表">
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

        {visibleProjects.map(({ project, items }) => {
          const expanded = model.searching || !collapsed.has(project.id);
          const hasActive = items.some((item) => item.id === activeId);
          const menuOpen = menu?.kind === "project" && menu.project.id === project.id;
          return (
            <section key={project.id} className="project-group">
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

        {!model.searching && visibleProjects.length === 0 && conversations.length > 0 && (
          <p className="sidebar-empty compact">还没有项目</p>
        )}

        {model.unfiled.map(([label, items]) => (
          <section key={label} className="conversation-group">
            <h2>{label}</h2>
            {items.map(renderConversation)}
          </section>
        ))}

        {model.searching && visibleProjects.length === 0 && unfiledCount === 0 && (
          <p className="sidebar-empty">没有匹配的对话</p>
        )}
        {!model.searching && conversations.length === 0 && projects.length === 0 && (
          <p className="sidebar-empty">还没有对话</p>
        )}
      </nav>

      <div className="sidebar-foot">
        <button type="button" className="foot-row" onClick={onSettings}>
          <Settings size={16} />
          <span>设置</span>
        </button>
      </div>

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
