import type { RequestStatus } from "./types";

/** Semantic color family — each maps to the status-* tokens in globals.css (light + dark). */
export type StatusFamily =
  | "neutral"
  | "pending"
  | "info"
  | "progress"
  | "success"
  | "warning"
  | "danger";

/** Badge tint classes per family (token-based, dark-mode aware). */
export const FAMILY_TINT: Record<StatusFamily, string> = {
  neutral: "bg-muted text-muted-foreground",
  pending: "bg-status-pending-bg text-status-pending-fg",
  info: "bg-status-info-bg text-status-info-fg",
  progress: "bg-status-progress-bg text-status-progress-fg",
  success: "bg-status-success-bg text-status-success-fg",
  warning: "bg-status-warning-bg text-status-warning-fg",
  danger: "bg-status-danger-bg text-status-danger-fg",
};

/** Workflow request statuses → color family. */
export const REQUEST_STATUS_FAMILY: Record<RequestStatus, StatusFamily> = {
  draft: "neutral",
  pending_dean: "pending",
  pending: "info",
  pending_vp: "progress",
  returned: "warning",
  approved: "success",
  rejected: "danger",
  verified: "success",
  not_eligible: "danger",
  awaiting_signatures: "progress",
  completed: "success",
  issued: "success",
};

/** Scientific-activity statuses → color family. */
export const ACTIVITY_STATUS_FAMILY: Record<string, StatusFamily> = {
  pending: "pending",
  approved: "success",
  returned: "warning",
  rejected: "danger",
};

/** Journal watchlist classifications → color family. */
export const SCREENING_FAMILY: Record<string, StatusFamily> = {
  accredited: "success",
  predatory: "danger",
  suspended: "warning",
};

/** Automatic journal check verdicts. "found" is informational, not a pass. */
export const JOURNAL_VERDICT_FAMILY: Record<string, StatusFamily> = {
  danger: "danger",
  caution: "warning",
  found: "info",
  unknown: "neutral",
};

const ACTIVE_STAGES: RequestStatus[] = [
  "pending_dean", "pending", "pending_vp", "verified", "awaiting_signatures",
];

/** Stages someone still has to act on; reviewers may change files only here (mirrors the backend). */
export function isActiveStage(status: RequestStatus): boolean {
  return ACTIVE_STAGES.includes(status);
}

export function requestTint(status: RequestStatus): string {
  return FAMILY_TINT[REQUEST_STATUS_FAMILY[status] ?? "neutral"];
}
