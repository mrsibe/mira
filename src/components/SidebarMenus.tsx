import { ArrowLeft, Check, ChevronRight, FolderInput, Pencil, Plus, Trash2 } from "lucide-react";
import { useState } from "react";
import { useMessages } from "../i18n";
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
  const t = useMessages();
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
      label={t.sidebarConversationActions()}
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
            <span>{t.sidebarEditTitle()}</span>
          </button>
          <button
            type="button"
            className="popup-menu-item"
            role="menuitem"
            onClick={() => setStep("move")}
          >
            <FolderInput size={15} />
            <span>{t.sidebarMoveCategory()}</span>
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
            <span>{t.commonDelete()}</span>
          </button>
        </>
      ) : (
        <>
          <div className="popup-menu-head">
            <button
              type="button"
              className="popup-menu-back"
              aria-label={t.sidebarBack()}
              title={t.sidebarBack()}
              onClick={() => setStep("root")}
            >
              <ArrowLeft size={15} />
            </button>
            <span>{t.sidebarMoveToProject()}</span>
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
              <span>{t.sidebarUnfiled()}</span>
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
            <span>{t.sidebarCreateProjectEllipsis()}</span>
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
  const t = useMessages();
  return (
    <PopupMenu
      anchor={anchor}
      label={t.sidebarProjectActionsLabel(project.name)}
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
        <span>{t.sidebarRenameProject()}</span>
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
        <span>{t.sidebarDeleteProject()}</span>
      </button>
    </PopupMenu>
  );
}
