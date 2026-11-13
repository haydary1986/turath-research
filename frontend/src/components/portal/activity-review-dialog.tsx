"use client";

import { useState, type ReactNode } from "react";
import { useTranslations } from "next-intl";
import { api } from "@/lib/api";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/portal/form-controls";
import { usePortalErrors } from "@/components/portal/use-portal-errors";
import { validateApproval, validateRefusal, type Activity } from "@/lib/portal/activity";
import { formatDate } from "@/lib/portal/types";

interface ActivityReviewDialogProps {
  activity: Activity;
  locale: string;
  onClose: () => void;
  onDecided: () => void;
}

function Detail({ label, wide, children }: { label: string; wide?: boolean; children: ReactNode }) {
  return (
    <div className={wide ? "sm:col-span-2" : undefined}>
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd className="font-medium whitespace-pre-line break-words">{children === null || children === "" ? "—" : children}</dd>
    </div>
  );
}

function FieldProblem({ id, message }: { id: string; message: string | null }) {
  if (!message) return null;
  return <p id={id} className="text-xs text-status-danger">{message}</p>;
}

export function ActivityReviewDialog({ activity: a, locale, onClose, onDecided }: ActivityReviewDialogProps) {
  const t = useTranslations("portal");
  const ta = useTranslations("portal.activities");
  const toMessage = usePortalErrors();
  const [approvalNo, setApprovalNo] = useState("");
  const [approvalNote, setApprovalNote] = useState("");
  const [note, setNote] = useState("");
  const [problems, setProblems] = useState<string[]>([]);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  function problemFor(field: string): string | null {
    const problem = problems.find((p) => p.startsWith(`${field}:`));
    return problem ? ta(`validation.${problem.replace(":", "_")}`) : null;
  }

  async function submit(status: string, body: { approval_no: string; note: string }, found: string[]) {
    setProblems(found);
    setError("");
    if (found.length > 0) return;
    setBusy(true);
    try {
      await api.post(`/api/v1/activities/${a.id}/decide`, { status, ...body });
      onDecided();
    } catch (err) {
      setError(toMessage(err));
    } finally {
      setBusy(false);
    }
  }

  const approve = () =>
    submit("approved", { approval_no: approvalNo, note: approvalNote }, validateApproval(approvalNo, approvalNote));
  const refuse = (status: "returned" | "rejected") =>
    submit(status, { approval_no: "", note }, validateRefusal(note));

  const approvalProblem = problemFor("approval_no");
  const approvalNoteProblem = problemFor("approval_note");
  const noteProblem = problemFor("note");

  return (
    <Dialog open onOpenChange={(open) => { if (!open && !busy) onClose(); }}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle className="pe-8">{ta("review.title")}</DialogTitle>
          <DialogDescription>{ta("review.intro")}</DialogDescription>
        </DialogHeader>

        <section aria-labelledby={`details-${a.id}`} className="space-y-3">
          <h3 id={`details-${a.id}`} className="font-semibold">{ta("review.details")}</h3>
          <dl className="grid gap-3 rounded-lg bg-muted/40 p-3 sm:grid-cols-2">
            <Detail label={ta("fields.title")} wide>{a.title_ar}</Detail>
            <Detail label={ta("fields.description")} wide>{a.description}</Detail>
            <Detail label={ta("fields.targetAudience")}>{a.target_audience}</Detail>
            <Detail label={ta("fields.organizer")}>{a.organizer}</Detail>
            <Detail label={ta("fields.attendanceMode")}>
              {a.attendance_mode ? ta(`modes.${a.attendance_mode}`) : null}
            </Detail>
            <Detail label={ta("fields.type")}>{ta(`types.${a.activity_type}`)}</Detail>
            <Detail label={ta("fields.date")}>{formatDate(a.activity_date, locale)}</Detail>
            <Detail label={ta("fields.location")}>{a.location}</Detail>
            <Detail label={ta("fields.presenter")}>{a.presenter}</Detail>
            <Detail label={ta("fields.attendees")}>{a.attendees_count}</Detail>
            <Detail label={t("fields.college")}>{a.college_name}</Detail>
            <Detail label={t("fields.department")}>{a.department_name}</Detail>
            <Detail label={ta("fields.registeredBy")} wide>{a.creator_name}</Detail>
            {a.decision_note && (
              <Detail label={ta("review.previousNote")} wide>{a.decision_note}</Detail>
            )}
          </dl>
        </section>

        {error && <Alert variant="danger">{error}</Alert>}

        <section className="space-y-3 rounded-lg border p-3">
          <div>
            <h3 className="font-semibold">{ta("review.approveTitle")}</h3>
            <p className="text-xs text-muted-foreground">{ta("review.approveHint")}</p>
          </div>
          <div className="space-y-1">
            <Label htmlFor={`no-${a.id}`}>{ta("fields.approvalNo")} *</Label>
            <Input
              id={`no-${a.id}`} dir="ltr" className="sm:w-56" value={approvalNo}
              placeholder={ta("approvalPlaceholder")}
              aria-invalid={!!approvalProblem}
              aria-describedby={approvalProblem ? `no-problem-${a.id}` : undefined}
              onChange={(e) => setApprovalNo(e.target.value)}
            />
            <FieldProblem id={`no-problem-${a.id}`} message={approvalProblem} />
          </div>
          <div className="space-y-1">
            <Label htmlFor={`approval-note-${a.id}`}>{ta("review.approvalNoteLabel")}</Label>
            <Input
              id={`approval-note-${a.id}`} value={approvalNote}
              aria-invalid={!!approvalNoteProblem}
              aria-describedby={approvalNoteProblem ? `approval-note-problem-${a.id}` : undefined}
              onChange={(e) => setApprovalNote(e.target.value)}
            />
            <FieldProblem id={`approval-note-problem-${a.id}`} message={approvalNoteProblem} />
          </div>
          <Button size="sm" disabled={busy} onClick={approve}>{ta("approve")}</Button>
        </section>

        <section className="space-y-3 rounded-lg border p-3">
          <div>
            <h3 className="font-semibold">{ta("review.refuseTitle")}</h3>
            <ul className="mt-1 space-y-1 text-xs text-muted-foreground">
              <li>{ta("review.returnHint")}</li>
              <li>{ta("review.rejectHint")}</li>
            </ul>
          </div>
          <div className="space-y-1">
            <Label htmlFor={`note-${a.id}`}>{ta("review.noteLabel")} *</Label>
            <Textarea
              id={`note-${a.id}`} value={note}
              aria-invalid={!!noteProblem}
              aria-describedby={noteProblem ? `note-problem-${a.id}` : undefined}
              onChange={(e) => setNote(e.target.value)}
            />
            <FieldProblem id={`note-problem-${a.id}`} message={noteProblem} />
          </div>
          <div className="flex flex-wrap gap-2">
            <Button size="sm" variant="outline" disabled={busy} onClick={() => refuse("returned")}>
              {ta("review.returnAction")}
            </Button>
            <Button size="sm" variant="destructive" disabled={busy} onClick={() => refuse("rejected")}>
              {ta("review.rejectAction")}
            </Button>
          </div>
        </section>
      </DialogContent>
    </Dialog>
  );
}
