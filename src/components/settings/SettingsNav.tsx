import { Brain, Cpu, Info, Palette, Search, Sparkles } from "lucide-react";
import type { SettingsSection } from "../../lib/types";

const sections = [
  { value: "general", label: "外观", icon: Palette },
  { value: "personalization", label: "个性化", icon: Sparkles },
  { value: "memory", label: "记忆", icon: Brain },
  { value: "search", label: "搜索", icon: Search },
  { value: "model", label: "模型", icon: Cpu },
  { value: "about", label: "关于", icon: Info },
] as const;

type SettingsNavProps = {
  section: SettingsSection;
  onChange: (section: SettingsSection) => void;
};

export function SettingsNav({ section, onChange }: SettingsNavProps) {
  return (
    <nav className="settings-nav" aria-label="设置栏目">
      <div className="settings-nav-list">
        {sections.map(({ value, label, icon: Icon }) => (
          <button
            key={value}
            type="button"
            className={`settings-nav-item ${section === value ? "active" : ""}`}
            aria-current={section === value ? "page" : undefined}
            onClick={() => onChange(value)}
          >
            <Icon size={16} />
            <span>{label}</span>
          </button>
        ))}
      </div>
    </nav>
  );
}
