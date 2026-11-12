import { Check, ChevronDown } from "lucide-react";
import { Fragment, type KeyboardEvent, useCallback, useState } from "react";
import { cx } from "./cx";
import { PopupMenu } from "./PopupMenu";

export type SelectOption<T extends string> = {
  value: T;
  label: string;
  previewFont?: string;
  group?: string;
};

type SelectProps<T extends string> = {
  label: string;
  value: T;
  options: ReadonlyArray<SelectOption<T>>;
  onChange: (value: T) => void;
  className?: string;
  disabled?: boolean;
};

export function Select<T extends string>({
  label,
  value,
  options,
  onChange,
  className,
  disabled,
}: SelectProps<T>) {
  const [anchor, setAnchor] = useState<HTMLButtonElement | null>(null);
  const selected = options.find((option) => option.value === value);

  const close = useCallback(() => {
    setAnchor(null);
    anchor?.focus();
  }, [anchor]);

  function onKeyDown(event: KeyboardEvent<HTMLButtonElement>) {
    if (event.key !== "ArrowDown") return;
    event.preventDefault();
    setAnchor(event.currentTarget);
  }

  return (
    <>
      <button
        type="button"
        className={cx("select-trigger", className)}
        aria-label={label}
        aria-haspopup="menu"
        aria-expanded={anchor !== null}
        disabled={disabled}
        onClick={(event) => setAnchor(anchor ? null : event.currentTarget)}
        onKeyDown={onKeyDown}
      >
        <span>{selected?.label ?? "选择"}</span>
        <ChevronDown size={15} aria-hidden="true" />
      </button>
      {anchor && (
        <PopupMenu anchor={anchor} label={label} className="select-menu" onClose={close}>
          {options.map((option, index) => {
            const active = option.value === value;
            const showGroup = option.group && option.group !== options[index - 1]?.group;
            return (
              <Fragment key={option.value}>
                {showGroup && (
                  <div className="select-group-label" role="presentation">
                    {option.group}
                  </div>
                )}
                <button
                  type="button"
                  className="popup-menu-item select-option"
                  role="menuitemradio"
                  aria-checked={active}
                  onClick={() => {
                    onChange(option.value);
                    close();
                  }}
                >
                  <span style={option.previewFont ? { fontFamily: option.previewFont } : undefined}>
                    {option.label}
                  </span>
                  {active && <Check size={15} className="select-check" aria-hidden="true" />}
                </button>
              </Fragment>
            );
          })}
        </PopupMenu>
      )}
    </>
  );
}
