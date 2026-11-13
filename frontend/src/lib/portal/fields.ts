import type { RequestKind } from "./types";

export interface FieldDef {
  key: string;
  type: "text" | "date" | "select" | "url" | "textarea";
  options?: string[];
  required?: boolean;
  ltr?: boolean;
}

export const PLAN_FIELDS: FieldDef[] = [
  { key: "work_type", type: "select", options: ["individual", "joint"], required: true },
  { key: "expected_completion", type: "date", required: true },
  { key: "participants", type: "textarea" },
  { key: "notes", type: "textarea" },
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
  return kind === "research_plan" ? PLAN_FIELDS : JOURNAL_FIELDS;
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
