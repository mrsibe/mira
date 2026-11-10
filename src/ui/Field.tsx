import { type ReactNode, useId } from "react";
import { cx } from "./cx";

type FieldProps = {
  label: string;
  children: (id: string) => ReactNode;
  className?: string;
};

export function Field({ label, className, children }: FieldProps) {
  const id = useId();

  return (
    <div className={cx("settings-field", className)}>
      <label htmlFor={id}>{label}</label>
      {children(id)}
    </div>
  );
}
