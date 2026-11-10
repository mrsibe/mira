import { PanelLeft } from "lucide-react";
import { IconButton } from "../ui";

type WindowTitlebarProps = {
  sidebarOpen: boolean;
  title: string;
  onToggleSidebar: () => void;
};

export function WindowTitlebar({ sidebarOpen, title, onToggleSidebar }: WindowTitlebarProps) {
  return (
    <header className="window-titlebar">
      <div className="window-titlebar-leading">
        <IconButton
          label={sidebarOpen ? "收起侧栏" : "展开侧栏"}
          className="window-titlebar-toggle"
          aria-expanded={sidebarOpen}
          onClick={onToggleSidebar}
        >
          <PanelLeft size={17} />
        </IconButton>
        <span className="window-titlebar-brand">Mira</span>
        {title && (
          <>
            <span className="window-titlebar-divider" aria-hidden="true" />
            <span className="window-titlebar-title">{title}</span>
          </>
        )}
      </div>
    </header>
  );
}
