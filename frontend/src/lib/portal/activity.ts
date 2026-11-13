export const MAX_APPROVAL_NO_LENGTH = 100;
export const MAX_DECISION_NOTE_LENGTH = 2000;

export interface Activity {
  id: string;
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
