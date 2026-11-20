import { Brain, Cpu, Info, Palette, Search, Sparkles } from "lucide-react";
import { useMessages } from "../../i18n";
import type { SettingsSection } from "../../lib/types";

const sections = [
  { value: "general", icon: Palette },
  { value: "personalization", icon: Sparkles },
  { value: "memory", icon: Brain },
  { value: "search", icon: Search },
  { value: "model", icon: Cpu },
  { value: "about", icon: Info },
] as const;

type SettingsNavProps = {
  section: SettingsSection;
  onChange: (section: SettingsSection) => void;
};

export function SettingsNav({ section, onChange }: SettingsNavProps) {
  const t = useMessages();
  const labels: Record<SettingsSection, string> = {
    general: t.settingsNavGeneral(),
    personalization: t.settingsNavPersonalization(),
    memory: t.settingsNavMemory(),
    search: t.settingsNavSearch(),
    model: t.settingsNavModel(),
    about: t.settingsNavAbout(),
  };
  return (
    <nav className="settings-nav" aria-label={t.settingsNavLabel()}>
      <div className="settings-nav-list">
        {sections.map(({ value, icon: Icon }) => (
          <button
            key={value}
            type="button"
            className={`settings-nav-item ${section === value ? "active" : ""}`}
            aria-current={section === value ? "page" : undefined}
            onClick={() => onChange(value)}
          >
            <Icon size={16} />
            <span>{labels[value]}</span>
          </button>
        ))}
      </div>
    </nav>
  );
}
