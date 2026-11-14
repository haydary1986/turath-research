/** Two-way profile sync with the RIS (ris.uoturath.edu.iq): admin API shapes. */

export interface RISSyncStats {
  ris_researchers: number;
  matched: number;
  unmatched_ris: number;
  to_ris: number;
  to_portal: number;
  agreed: number;
  reviews: number;
  skipped: number;
  failed: number;
  unmapped_departments: string[] | null;
}

export interface RISSyncRun {
  id: string;
  mode: "dry_run" | "apply";
  trigger: "schedule" | "manual";
  started_at: string;
  finished_at: string | null;
  stats: Partial<RISSyncStats>;
  failed: boolean;
}

export interface RISSyncStatus {
  configured: boolean;
  scheduled: boolean;
  interval_minutes: number;
  open_reviews: number;
  linked_profiles: number;
  runs: RISSyncRun[];
}

export interface RISSyncChange {
  researcher_id: string | null;
  researcher_name: string | null;
  field: string;
  action: "to_ris" | "to_portal" | "review" | "agree";
  old_value: string | null;
  new_value: string | null;
  applied: boolean;
  note: string | null;
}

export interface RISSyncReview {
  id: string;
  researcher_id: string;
  researcher_name: string;
  field: string;
  portal_value: string;
  ris_value: string;
  reason: "conflict" | "taken";
  created_at: string;
}

export const RIS_SYNC_API = {
  status: "/api/v1/admin/ris-sync",
  runs: "/api/v1/admin/ris-sync/runs",
  changes: (runId: string): string => `/api/v1/admin/ris-sync/runs/${encodeURIComponent(runId)}/changes`,
  reviews: "/api/v1/admin/ris-sync/reviews",
  resolve: (id: string): string => `/api/v1/admin/ris-sync/reviews/${encodeURIComponent(id)}/resolve`,
} as const;

export const RIS_SYNC_FIELDS = [
  "name_ar", "name_en", "department", "phone",
  "orcid", "google_scholar", "scopus", "researchgate", "linkedin", "webofscience",
] as const;

/** Values that are links open in a new tab; others are shown as text. */
export function isLink(value: string | null | undefined): boolean {
  return !!value && /^https:\/\//.test(value);
}
