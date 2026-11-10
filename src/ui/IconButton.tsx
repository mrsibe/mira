import type { ButtonHTMLAttributes } from "react";
import { cx } from "./cx";

type IconButtonProps = Omit<ButtonHTMLAttributes<HTMLButtonElement>, "aria-label" | "title"> & {
  label: string;
};

export function IconButton({ label, className, type = "button", ...props }: IconButtonProps) {
  return (
    <button
      type={type}
      className={cx("icon-button", className)}
      aria-label={label}
      title={label}
      {...props}
    />
  );
}
