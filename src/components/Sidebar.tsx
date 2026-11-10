import { Pencil, Search, Settings, SquarePen, Trash2, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { groupConversations } from "../lib/format";
import type { Conversation } from "../mygo";

type SidebarProps = {
  conversations: Conversation[];
  activeId: string | null;
  search: string;
  onSearch: (value: string) => void;
  onSelect: (id: string) => void;
  onNew: () => void;
  onRename: (id: string, title: string) => void;
  onDelete: (id: string) => void;
  onSettings: () => void;
};

export function Sidebar({
  conversations,
  activeId,
  search,
  onSearch,
  onSelect,
  onNew,
  onRename,
  onDelete,
  onSettings,
}: SidebarProps) {
  const [editingId, setEditingId] = useState<string | null>(null);
  const [draft, setDraft] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (editingId) inputRef.current?.focus();
  }, [editingId]);

  const filtered = conversations.filter((item) => {
    const query = search.trim().toLocaleLowerCase();
    if (!query) return true;
    return (
      item.title.toLocaleLowerCase().includes(query) ||
      item.preview.toLocaleLowerCase().includes(query)
    );
  });

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

      <nav className="conversation-list" aria-label="对话列表">
        {filtered.length === 0 ? (
          <p className="sidebar-empty">{search ? "没有匹配的对话" : "还没有对话"}</p>
        ) : (
          groupConversations(filtered).map(([label, items]) => (
            <section key={label} className="conversation-group">
              <h2>{label}</h2>
              {items.map((item) => {
                const active = item.id === activeId;
                const editing = item.id === editingId;
                return (
                  <div key={item.id} className={`conversation-item ${active ? "active" : ""}`}>
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
                          title="重命名"
                          onClick={(event) => {
                            event.stopPropagation();
                            startRename(item);
                          }}
                        >
                          <Pencil size={13} />
                        </button>
                        <button
                          type="button"
                          className="mini-button danger"
                          title="删除"
                          onClick={(event) => {
                            event.stopPropagation();
                            onDelete(item.id);
                          }}
                        >
                          <Trash2 size={13} />
                        </button>
                      </span>
                    )}
                  </div>
                );
              })}
            </section>
          ))
        )}
      </nav>

      <div className="sidebar-foot">
        <button type="button" className="foot-row" onClick={onSettings}>
          <Settings size={16} />
          <span>设置</span>
        </button>
      </div>
    </aside>
  );
}
