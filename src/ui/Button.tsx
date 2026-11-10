import type { ButtonHTMLAttributes } from "react";
import { cx } from "./cx";

export type ButtonVariant = "primary" | "secondary" | "danger" | "quiet" | "dangerText";

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: ButtonVariant;
};

const variantClasses: Record<ButtonVariant, string | undefined> = {
  primary: "primary",
  secondary: "secondary",
  danger: "danger",
  quiet: undefined,
  dangerText: "text-danger",
};

export function Button({
  variant = "secondary",
  className,
  type = "button",
  ...props
}: ButtonProps) {
  return (
    <button type={type} className={cx("button", variantClasses[variant], className)} {...props} />
  );
}
