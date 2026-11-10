import type { ReactNode } from "react";
import { cx } from "./cx";

type NoticeProps = {
  children: ReactNode;
  tone?: "error" | "warning";
  className?: string;
};

export function Notice({ children, tone = "warning", className }: NoticeProps) {
  return (
    <p
      className={cx(tone === "error" ? "form-error" : "form-notice", className)}
      role={tone === "error" ? "alert" : "status"}
    >
      {children}
    </p>
  );
}
