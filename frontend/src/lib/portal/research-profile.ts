export interface ProfilePublication {
  id: string;
  title: string;
  journal: string | null;
  year: number | null;
  doi: string | null;
  citation_count: number;
}

export interface ResearchProfile {
  linked: boolean;
  google_scholar_url: string | null;
  scopus_url: string | null;
  researchgate_url: string | null;
  linkedin_url: string | null;
  web_of_science_url: string | null;
  phone: string | null;
  phone_secondary: string | null;
  recovery_email: string | null;
  orcid: string | null;
  orcid_url: string | null;
  openalex_id: string | null;
  h_index: number;
  h_index_source: string | null;
  total_citations: number;
  total_publications: number;
  last_synced_at: string | null;
  last_sync_error: string | null;
  syncing: boolean;
  publications: ProfilePublication[];
}

export interface ResearcherLinks {
  google_scholar_url: string | null;
  scopus_url: string | null;
  researchgate_url: string | null;
  linkedin_url: string | null;
  web_of_science_url: string | null;
  orcid_url: string | null;
  last_synced_at: string | null;
  last_sync_error: string | null;
}

const IDENTITY_REVIEW_CODES = ["orcid_name_mismatch", "orcid_name_unverified", "openalex_name_mismatch"];

/** True when the last sync refused a source on a name check, which staff should look at. */
export function needsIdentityReview(code: string | null | undefined): boolean {
  return !!code && IDENTITY_REVIEW_CODES.includes(code);
}

export const PROFILE_ENDPOINT = "/api/v1/me/research-profile";

/** Personal data: the API answers only the researcher, scientific affairs, the VP and admin. */
export interface ResearcherContact {
  email: string;
  phone: string | null;
  phone_secondary: string | null;
  recovery_email: string | null;
  employee_no: string | null;
  account_status: string | null;
}

export type ContactKind = "tel" | "mailto" | "text";

export interface ContactEntry {
  key: "contactEmail" | "phone" | "phone_secondary" | "recovery_email" | "contactEmployeeNo";
  value: string;
  kind: ContactKind;
}

/** The filled rows of the contact card, in display order. */
export function contactEntries(contact: ResearcherContact): ContactEntry[] {
  const rows: (Omit<ContactEntry, "value"> & { value: string | null })[] = [
    { key: "contactEmail", value: contact.email, kind: "mailto" },
    { key: "phone", value: contact.phone, kind: "tel" },
    { key: "phone_secondary", value: contact.phone_secondary, kind: "tel" },
    { key: "recovery_email", value: contact.recovery_email, kind: "mailto" },
    { key: "contactEmployeeNo", value: contact.employee_no, kind: "text" },
  ];
  return rows.flatMap((row) => (row.value ? [{ ...row, value: row.value }] : []));
}

export const BULK_SYNC_ENDPOINT = "/api/v1/admin/sync/orcid";

export interface BulkSyncRun {
  id: string;
  status: "running" | "waiting" | "stopped" | "completed";
  only_failed: boolean;
  processed: number;
  succeeded: number;
  failed: number;
  skipped: number;
  failures: Record<string, number>;
  wait_reason: string | null;
  resume_at: string | null;
}

export interface BulkSyncProgress {
  run: BulkSyncRun | null;
  eligible: number;
  remaining: number;
  active: boolean;
}
