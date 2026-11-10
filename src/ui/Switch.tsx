import { Check } from "lucide-react";

type SwitchProps = {
  checked: boolean;
  title: string;
  description: string;
  disabled?: boolean;
  onChange: () => void;
};

export function Switch({ checked, title, description, disabled = false, onChange }: SwitchProps) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      className="settings-switch"
      disabled={disabled}
      onClick={onChange}
    >
      <span className="settings-switch-copy">
        <strong>{title}</strong>
        <span>{description}</span>
      </span>
      <span className="settings-switch-track" aria-hidden="true">
        <span>{checked && <Check size={10} />}</span>
      </span>
    </button>
  );
}
