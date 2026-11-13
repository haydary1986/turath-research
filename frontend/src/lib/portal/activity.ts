export const MAX_APPROVAL_NO_LENGTH = 100;
export const MAX_DECISION_NOTE_LENGTH = 2000;

/** Categories of the ministry form FS 003, in the order of its sheets (the form has no sheet 8). */
export const ACTIVITY_CATEGORIES = [
  "infrastructure",
  "workshops",
  "courses",
  "local_research",
  "indexed_research",
  "international_research",
  "books",
  "patents",
  "conferences",
  "international_agreements",
  "local_agreements",
  "fellowships",
  "community_service",
] as const;

export type ActivityCategory = (typeof ACTIVITY_CATEGORIES)[number];

/** Entered for the institution: only responsibility holders and scientific affairs. Mirrors the backend. */
const INSTITUTIONAL: readonly string[] = ["infrastructure", "international_agreements", "local_agreements"];
/** The form column of these sheets is the venue, so it doubles as the activity's location. */
const PLACE_CATEGORIES: readonly string[] = ["workshops", "courses", "conferences", "community_service"];
const DETAIL_REQUIRED: readonly string[] = [
  "local_research", "indexed_research", "international_research", "books", "patents",
  "fellowships", "international_agreements", "local_agreements",
];
/** Sub-types kept inside the workshops category. */
export const WORKSHOP_TYPES = ["seminar", "workshop", "panel", "lecture"] as const;
const REGISTRAR_ROLES: readonly string[] = ["admin", "scientific_affairs"];
const EXPORT_ROLES: readonly string[] = ["admin", "scientific_affairs", "vp_research"];

/** One data-entry responsibility; a null college means the whole university. */
export interface Responsibility {
  id?: string;
  category: string;
  college_id: string | null;
  college_name?: string | null;
}

export function isCategory(value: string): value is ActivityCategory {
  return (ACTIVITY_CATEGORIES as readonly string[]).includes(value);
}

export function isInstitutional(category: string): boolean {
  return INSTITUTIONAL.includes(category);
}

export function isPlaceCategory(category: string): boolean {
  return PLACE_CATEGORIES.includes(category);
}

export function isDetailRequired(category: string): boolean {
  return DETAIL_REQUIRED.includes(category);
}

/** Whether the responsibilities cover the category at the college (null: university level). */
export function holds(scopes: readonly Responsibility[], category: string, collegeId: string | null): boolean {
  return scopes.some(
    (s) => s.category === category && (s.college_id === null || (collegeId !== null && s.college_id === collegeId))
  );
}

/** Categories the user may choose when filing a record; mirrors activity.CanEnter for some college. */
export function enterableCategories(role: string | undefined, scopes: readonly Responsibility[]): ActivityCategory[] {
  return ACTIVITY_CATEGORIES.filter(
    (c) => !isInstitutional(c) || (role !== undefined && REGISTRAR_ROLES.includes(role)) || scopes.some((s) => s.category === c)
  );
}

/** Whether the user may download the FS 003 form (the backend then limits the rows to their scope). */
export function canExportFS003(role: string | undefined, scopes: readonly Responsibility[]): boolean {
  return (role !== undefined && EXPORT_ROLES.includes(role)) || scopes.length > 0;
}

/** The download name the backend gives the FS 003 form for this period. */
export function fs003Filename(from: string, to: string): string {
  return `fs003-${from}_${to}.xlsx`;
}

export interface Activity {
  id: string;
  category: string;
  category_detail: string | null;
  item_count: number | null;
  notes: string | null;
  can_edit: boolean;
  activity_type: string;
  title_ar: string;
  description: string | null;
  presenter: string | null;
  college_id: string | null;
  department_id: string | null;
  college_name: string | null;
  department_name: string | null;
  activity_date: string;
  location: string | null;
  attendees_count: number;
  target_audience: string | null;
  organizer: string | null;
  attendance_mode: string | null;
  status: string;
  approval_no: string | null;
  decision_note: string | null;
  created_by: string;
  creator_name: string;
}

function runeLength(value: string): number {
  return [...value].length;
}

/** Message key under portal.activities.statuses: a returned activity addresses its owner directly. */
export function activityStatusKey(status: string, viewerIsOwner: boolean): string {
  return status === "returned" && viewerIsOwner ? "returnedToYou" : status;
}

/**
 * The note on the row. A resubmitted activity is pending again but keeps the
 * earlier return note in the database, and that note no longer describes its state.
 */
export function currentDecisionNote(status: string, note: string | null): string | null {
  return status === "pending" ? null : note;
}

/** "field:code" problems for approving; mirrors the backend's DecideActivity rules. */
export function validateApproval(approvalNo: string, note: string): string[] {
  const number = approvalNo.trim();
  return [
    ...(number === "" ? ["approval_no:required"] : []),
    ...(runeLength(number) > MAX_APPROVAL_NO_LENGTH ? ["approval_no:too_long"] : []),
    ...(runeLength(note.trim()) > MAX_DECISION_NOTE_LENGTH ? ["approval_note:too_long"] : []),
  ];
}

/** "field:code" problems for returning or rejecting; mirrors the backend's DecideActivity rules. */
export function validateRefusal(note: string): string[] {
  const trimmed = note.trim();
  return [
    ...(trimmed === "" ? ["note:required"] : []),
    ...(runeLength(trimmed) > MAX_DECISION_NOTE_LENGTH ? ["note:too_long"] : []),
  ];
}

/** Message key under portal.activities.dateLabels: the form names the date column per sheet. */
export function dateLabelKey(category: string): string {
  switch (category) {
    case "local_research":
    case "indexed_research":
    case "international_research":
      return "research";
    case "international_agreements":
    case "local_agreements":
      return "agreements";
    case "books":
    case "patents":
    case "conferences":
    case "fellowships":
      return category;
    default:
      return "default";
  }
}

/** Whether the sheet describes an event with presenters and attendance (workshops, courses…). */
export function isEventCategory(category: string): boolean {
  return isPlaceCategory(category);
}
