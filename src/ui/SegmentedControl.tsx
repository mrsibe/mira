import { type ComponentType, useId } from "react";
import { cx } from "./cx";

type SegmentOption<T extends string> = {
  value: T;
  label: string;
  icon?: ComponentType<{ size?: number }>;
};

type SegmentedControlProps<T extends string> = {
  label: string;
  value: T;
  options: ReadonlyArray<SegmentOption<T>>;
  onChange: (value: T) => void;
  className?: string;
  optionClassName?: string;
};

export function SegmentedControl<T extends string>({
  label,
  value,
  options,
  onChange,
  className,
  optionClassName,
}: SegmentedControlProps<T>) {
  const name = useId();

  return (
    <fieldset className={cx("theme-options", className)} aria-label={label}>
      {options.map((option) => {
        const Icon = option.icon;
        const selected = value === option.value;
        return (
          <label
            key={option.value}
            className={cx("segmented-option", optionClassName, selected && "selected")}
          >
            <input
              className="sr-only"
              type="radio"
              name={name}
              value={option.value}
              checked={selected}
              onChange={() => onChange(option.value)}
            />
            {Icon && <Icon size={15} />}
            <span>{option.label}</span>
          </label>
        );
      })}
    </fieldset>
  );
}
