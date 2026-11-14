import type { ComponentProps } from "react";
import { cx } from "./cx";

export function Input({ className, ...props }: ComponentProps<"input">) {
  return <input className={cx("input", className)} {...props} />;
}
