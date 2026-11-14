import type { RequestKind } from "./types";

export interface FieldDef {
  key: string;
  type: "text" | "date" | "select" | "url" | "textarea";
  options?: string[];
  required?: boolean;
  ltr?: boolean;
}

/** A joint research plan names at most this many partners (mirrors the backend). */
export const MAX_CO_RESEARCHERS = 3;

/** Detail keys holding the co-researchers of a joint plan: co_researcher_1 … co_researcher_3. */
export const CO_RESEARCHER_KEYS: string[] = Array.from(
  { length: MAX_CO_RESEARCHERS },
  (_, i) => `co_researcher_${i + 1}`
);

export const PLAN_FIELDS: FieldDef[] = [
  { key: "work_type", type: "select", options: ["individual", "joint"], required: true },
  { key: "expected_completion", type: "date", required: true },
  { key: "notes", type: "textarea" },
];

export const TITLE_CHANGE_FIELDS: FieldDef[] = [
  { key: "old_title", type: "textarea", required: true },
  { key: "reason", type: "textarea", required: true },
];

export const JOURNAL_FIELDS: FieldDef[] = [
  { key: "journal_name", type: "text", required: true },
  { key: "publisher", type: "text", required: true },
  { key: "issn", type: "text", ltr: true },
  { key: "issn_online", type: "text", ltr: true },
  {
    key: "index_db",
    type: "select",
    options: ["scopus", "wos", "local", "international_unindexed"],
    required: true,
  },
  { key: "quartile", type: "select", options: ["Q1", "Q2", "Q3", "Q4"] },
  { key: "cite_score", type: "text", ltr: true },
  { key: "impact_factor", type: "text", ltr: true },
  { key: "submission_date", type: "date" },
  { key: "acceptance_date", type: "date" },
  { key: "publication_date", type: "date", required: true },
  { key: "volume", type: "text", ltr: true },
  { key: "issue", type: "text", ltr: true },
  { key: "pages", type: "text", ltr: true },
  { key: "authors_count", type: "text", ltr: true },
  { key: "author_order", type: "text", ltr: true },
  {
    key: "extracted_thesis",
    type: "select",
    options: ["not_extracted", "extracted_master", "extracted_phd"],
  },
  { key: "plagiarism_pct", type: "text", ltr: true },
  { key: "article_url", type: "url", ltr: true },
  { key: "journal_url", type: "url", ltr: true },
  { key: "index_url", type: "url", ltr: true },
  { key: "author_url", type: "url", ltr: true },
  { key: "co_authors", type: "textarea" },
  { key: "notes", type: "textarea" },
];

export function fieldsFor(kind: RequestKind): FieldDef[] {
  if (kind === "research_plan") return PLAN_FIELDS;
  if (kind === "title_change") return TITLE_CHANGE_FIELDS;
  return JOURNAL_FIELDS;
}

/** Whether the request concerns a journal publication (and so gets the journal check and a DOI). */
export function hasJournalData(kind: RequestKind): boolean {
  return kind === "journal_verification" || kind === "appreciation_letter";
}

/** Non-blank co-researcher names of a plan, in order. */
export function coResearchers(details: Record<string, string>): string[] {
  return CO_RESEARCHER_KEYS.map((key) => details[key]?.trim() ?? "").filter(Boolean);
}

/**
 * Applies a work-type change: an individual plan drops every co-researcher, so the saved
 * draft never carries names the backend would reject.
 */
export function withWorkType(details: Record<string, string>, workType: string): Record<string, string> {
  const next = { ...details, work_type: workType };
  if (workType === "joint") return next;
  return Object.fromEntries(Object.entries(next).filter(([key]) => !CO_RESEARCHER_KEYS.includes(key)));
}

/** Only these indexing databases rank journals in quartiles (mirrors the backend). */
export function quartileApplies(indexDb: string | undefined): boolean {
  return indexDb === "scopus" || indexDb === "wos";
}

/** Drops a quartile the chosen indexing database cannot have. */
export function withoutStaleQuartile(details: Record<string, string>): Record<string, string> {
  if (quartileApplies(details.index_db) || !("quartile" in details)) return details;
  return Object.fromEntries(Object.entries(details).filter(([key]) => key !== "quartile"));
}

/** Attachment roles that must be present before submitting (mirrors the backend rules). */
export function requiredRoles(kind: RequestKind, hasRelatedVerification: boolean): string[] {
  if (kind === "research_plan") return ["research_form"];
  if (kind !== "appreciation_letter") return [];
  return hasRelatedVerification ? ["committee_approval"] : ["journal_form", "committee_approval"];
}
