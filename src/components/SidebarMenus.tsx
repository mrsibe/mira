import { ArrowLeft, Check, ChevronRight, FolderInput, Pencil, Plus, Trash2 } from "lucide-react";
import { useState } from "react";
import type { Conversation, Project } from "../mygo";
import { PopupMenu } from "../ui";

type ConversationMenuProps = {
  conversation: Conversation;
  projects: Project[];
  anchor: HTMLElement;
  onClose: () => void;
  onRename: () => void;
  onMove: (projectID: string) => Promise<boolean>;
  onDelete: () => void;
  onCreateProject: () => void;
};

export function ConversationMenu({
  conversation,
  projects,
  anchor,
  onClose,
  onRename,
  onMove,
  onDelete,
  onCreateProject,
}: ConversationMenuProps) {
  const [step, setStep] = useState<"root" | "move">("root");
  const [busy, setBusy] = useState(false);
  const knownProject = projects.some((project) => project.id === conversation.projectId);
  const currentProjectID = knownProject ? (conversation.projectId ?? "") : "";

  async function move(projectID: string) {
    if (busy) return;
    setBusy(true);
    const moved = await onMove(projectID);
    setBusy(false);
    if (moved) onClose();
  }

  return (
    <PopupMenu
      key={step}
      anchor={anchor}
      label="对话操作"
      className="sidebar-popup"
      onClose={onClose}
    >
      {step === "root" ? (
        <>
          <button
            type="button"
            className="popup-menu-item"
            role="menuitem"
            onClick={() => {
              onClose();
              onRename();
            }}
          >
            <Pencil size={15} />
            <span>编辑标题</span>
          </button>
          <button
            type="button"
            className="popup-menu-item"
            role="menuitem"
            onClick={() => setStep("move")}
          >
            <FolderInput size={15} />
            <span>移动分类</span>
            <ChevronRight size={15} className="popup-menu-trailing" />
          </button>
          <div className="popup-menu-separator" />
          <button
            type="button"
            className="popup-menu-item danger"
            role="menuitem"
            onClick={() => {
              onClose();
              onDelete();
            }}
          >
            <Trash2 size={15} />
            <span>删除</span>
          </button>
        </>
      ) : (
        <>
          <div className="popup-menu-head">
            <button
              type="button"
              className="popup-menu-back"
              aria-label="返回"
              title="返回"
              onClick={() => setStep("root")}
            >
              <ArrowLeft size={15} />
            </button>
            <span>移动到项目</span>
          </div>
          <div className="popup-menu-scroll">
            <button
              type="button"
              className="popup-menu-item"
              role="menuitemradio"
              aria-checked={currentProjectID === ""}
              disabled={busy}
              onClick={() => void move("")}
            >
              <span>未分类</span>
              {currentProjectID === "" && <Check size={15} className="popup-menu-trailing" />}
            </button>
            {projects.map((project) => {
              const selected = currentProjectID === project.id;
              return (
                <button
                  key={project.id}
                  type="button"
                  className="popup-menu-item"
                  role="menuitemradio"
                  aria-checked={selected}
                  disabled={busy}
                  onClick={() => void move(project.id)}
                >
                  <span>{project.name}</span>
                  {selected && <Check size={15} className="popup-menu-trailing" />}
                </button>
              );
            })}
          </div>
          <div className="popup-menu-separator" />
          <button
            type="button"
            className="popup-menu-item"
            role="menuitem"
            onClick={() => {
              onClose();
              onCreateProject();
            }}
          >
            <Plus size={15} />
            <span>新建项目…</span>
          </button>
        </>
      )}
    </PopupMenu>
  );
}

type ProjectMenuProps = {
  project: Project;
  anchor: HTMLElement;
  onClose: () => void;
  onRename: () => void;
  onDelete: () => void;
};

export function ProjectMenu({ project, anchor, onClose, onRename, onDelete }: ProjectMenuProps) {
  return (
    <PopupMenu
      anchor={anchor}
      label={`${project.name} 项目操作`}
      className="sidebar-popup"
      onClose={onClose}
    >
      <button
        type="button"
        className="popup-menu-item"
        role="menuitem"
        onClick={() => {
          onClose();
          onRename();
        }}
      >
        <Pencil size={15} />
        <span>重命名项目</span>
      </button>
      <div className="popup-menu-separator" />
      <button
        type="button"
        className="popup-menu-item danger"
        role="menuitem"
        onClick={() => {
          onClose();
          onDelete();
        }}
      >
        <Trash2 size={15} />
        <span>删除项目</span>
      </button>
    </PopupMenu>
  );
}
