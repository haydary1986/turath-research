import type { ReactNode } from "react";
import { cn } from "@/lib/utils";
import { FAMILY_TINT, type StatusFamily } from "@/lib/portal/status";

const BORDER: Record<StatusFamily, string> = {
  neutral: "border-border",
  pending: "border-status-pending/30",
  info: "border-status-info/30",
  progress: "border-status-progress/30",
  success: "border-status-success/30",
  warning: "border-status-warning/30",
  danger: "border-status-danger/30",
};

interface AlertProps {
  variant?: StatusFamily;
  className?: string;
  children: ReactNode;
}

/** Inline notice box (errors, success confirmations, workflow banners). */
export function Alert({ variant = "info", className, children }: AlertProps) {
  return (
    <div
      role="alert"
      className={cn(
        "rounded-lg border p-3 text-sm whitespace-pre-line",
        FAMILY_TINT[variant],
        BORDER[variant],
        className
      )}
    >
      {children}
    </div>
  );
}
