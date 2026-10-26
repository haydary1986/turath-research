export type RequestKind =
  | "research_plan"
  | "journal_verification"
  | "appreciation_letter";

export const REQUEST_KINDS: RequestKind[] = [
  "research_plan",
  "journal_verification",
  "appreciation_letter",
];

export type RequestStatus =
  | "draft"
  | "pending_dean"
  | "pending"
  | "pending_vp"
  | "returned"
  | "approved"
  | "rejected"
  | "verified"
  | "not_eligible"
  | "awaiting_signatures"
  | "completed"
  | "issued";

export interface PortalRequest {
  id: string;
  request_no: string | null;
  kind: RequestKind;
  status: RequestStatus;
  applicant_id: string;
  applicant_name: string;
  applicant_email: string;
  title_ar: string | null;
  title_en: string | null;
  doi: string | null;
  college_id: string | null;
  department_id: string | null;
  college_name: string | null;
  department_name: string | null;
  details: Record<string, string>;
  related_request_id: string | null;
  decision_note: string | null;
  sa_decision: string | null;
  submitted_at: string | null;
  decided_at: string | null;
  created_at: string;
}

export interface Attachment {
  id: string;
  role: string;
  original_name: string;
  mime_type: string;
  size_bytes: number;
  created_at: string;
}

export interface RequestEvent {
  from_status: string | null;
  to_status: string;
  note: string | null;
  actor_name: string | null;
  created_at: string;
}

export interface RequestDetail {
  request: PortalRequest;
  attachments: Attachment[];
  events: RequestEvent[];
  allowed_transitions: RequestStatus[];
  can_edit: boolean;
  letter_id: string | null;
}

export interface StatCount {
  kind: RequestKind;
  status: RequestStatus;
  count: number;
}

export interface LetterTemplate {
  id: string;
  name: string;
  header_line_1: string;
  header_line_2: string;
  title_text: string;
  body_text: string;
  closing_text: string;
  signer_name: string;
  signer_title: string;
  background_file_id: string | null;
  signature_file_id: string | null;
  is_active: boolean;
}

export interface IssuedLetter {
  id: string;
  request_id: string;
  letter_no: string;
  issued_date: string;
  recipient_name: string;
  college_name: string;
  department_name: string;
  header_line_1: string;
  header_line_2: string;
  title_text: string;
  body_text: string;
  closing_text: string;
  signer_name: string;
  signer_title: string;
  background_file_id: string | null;
  signature_file_id: string | null;
}

export interface Org {
  id: string;
  name_ar: string;
  name_en?: string | null;
  college_id?: string;
}

const STAFF_ROLES = ["admin", "scientific_affairs", "vp_research"];

export function isStaffRole(role?: string | null): boolean {
  return !!role && STAFF_ROLES.includes(role);
}

/** Roles of files the applicant must/can attach, per request kind. */
export const APPLICANT_ATTACHMENT_ROLES: Record<RequestKind, string[]> = {
  research_plan: ["research_form", "other"],
  journal_verification: ["journal_form", "committee_approval", "published_paper", "other"],
  appreciation_letter: ["journal_form", "committee_approval", "published_paper", "other"],
};

export const STAFF_ATTACHMENT_ROLES: Record<RequestKind, string[]> = {
  research_plan: ["signed_form", "other"],
  journal_verification: ["signed_form", "other"],
  appreciation_letter: ["other"],
};

/** Parses the API's "validation:field:code,field:code" error into field/code pairs, or null for other errors. */
export function parseValidation(message: string): { field: string; code: string }[] | null {
  if (!message.startsWith("validation:")) return null;
  return message
    .slice("validation:".length)
    .split(",")
    .filter(Boolean)
    .map((entry) => {
      const [field, code] = entry.split(":");
      return { field, code: code ?? "invalid" };
    });
}

export function formatDate(value: string | null | undefined, locale: string): string {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleDateString(locale === "ar" ? "ar-IQ-u-nu-latn" : "en-GB");
}

export function requestTitle(r: Pick<PortalRequest, "title_ar" | "title_en">): string {
  return r.title_en || r.title_ar || "—";
}
