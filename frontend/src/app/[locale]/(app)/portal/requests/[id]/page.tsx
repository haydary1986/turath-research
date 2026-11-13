"use client";

import { useParams } from "next/navigation";
import { useState } from "react";
import { useTranslations } from "next-intl";
import { Link } from "@/i18n/routing";
import { CheckCircle2, Clock, Printer } from "lucide-react";
import { api } from "@/lib/api";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { AttachmentsPanel } from "@/components/portal/attachments-panel";
import { JournalEvidencePanel } from "@/components/portal/journal-evidence-panel";
import { journalCheckParams, useJournalCheck } from "@/components/portal/use-journal-check";
import { RequireAuth } from "@/components/portal/require-auth";
import { OfficialFormButton } from "@/components/portal/official-form";
import { RequestStepper } from "@/components/portal/request-stepper";
import { RequestSummary } from "@/components/portal/request-summary";
import { ReviewActions } from "@/components/portal/review-actions";
import { StatusBadge } from "@/components/portal/status-badge";
import { Timeline } from "@/components/portal/timeline";
import { usePortalErrors } from "@/components/portal/use-portal-errors";
import { useRequestDetail } from "@/components/portal/use-request-detail";
import { requiredRoles } from "@/lib/portal/fields";
import { isActiveStage } from "@/lib/portal/status";
import {
  APPLICANT_ATTACHMENT_ROLES,
  STAFF_ATTACHMENT_ROLES,
  isStaffRole,
  requestTitle,
  type JournalCheck,
  type PortalRequest,
  type RequestDetail,
} from "@/lib/portal/types";
import { useAuth } from "@/lib/auth";

function DetailSkeleton() {
  return (
    <div className="space-y-6">
      <Skeleton className="h-8 w-2/3" />
      <Skeleton className="h-20 w-full" />
      <div className="grid gap-6 lg:grid-cols-3">
        <div className="space-y-6 lg:col-span-2">
          <Skeleton className="h-64 w-full" />
          <Skeleton className="h-40 w-full" />
        </div>
        <Skeleton className="h-64 w-full" />
      </div>
    </div>
  );
}

/** "Who has it now" notice for viewers with nothing to do at this stage. */
function WaitingNotice({ detail }: { detail: RequestDetail }) {
  const t = useTranslations("portal");
  const { status } = detail.request;
  const waiting = ["pending_dean", "pending", "pending_vp", "awaiting_signatures", "verified"];
  if (!waiting.includes(status)) return null;
  return (
    <Card>
      <CardContent className="flex items-center gap-3 pt-5 text-sm text-muted-foreground">
        <Clock className="size-5 shrink-0" aria-hidden />
        <span>{t(`statuses.${status}`)}</span>
      </CardContent>
    </Card>
  );
}

function RequestPage() {
  const params = useParams<{ id: string }>();
  const t = useTranslations("portal");
  const toMessage = usePortalErrors();
  const { user } = useAuth();
  const { detail, loading, notFound, reload } = useRequestDetail(params.id);
  const [submitting, setSubmitting] = useState(false);
  const [justSubmitted, setJustSubmitted] = useState(false);
  const [error, setError] = useState("");

  if (loading) return <DetailSkeleton />;
  if (notFound || !detail) {
    return <p className="text-center text-muted-foreground">{t("ui.notFound")}</p>;
  }

  const { request } = detail;
  const isApplicant = user?.id === request.applicant_id;
  const isReviewer = isStaffRole(user?.role) && !isApplicant;
  const reviewerMayChangeFiles = isReviewer && isActiveStage(request.status);
  const canAct = !isApplicant && (detail.allowed_transitions.length > 0 || detail.can_issue_letter);
  const required = requiredRoles(request.kind, !!request.related_request_id);
  const present = new Set(detail.attachments.map((a) => a.role));
  const missingRequired = required.filter((role) => !present.has(role));
  // Applicants print their own draft too: the plan form is signed on paper before submitting.
  const canPrint = !["draft", "returned"].includes(request.status) || isApplicant;
  const letterIssued = !!detail.letter_id;

  async function submit() {
    setError("");
    setSubmitting(true);
    try {
      await api.post(`/api/v1/portal/requests/${request.id}/submit`, {});
      setJustSubmitted(true);
      reload();
    } catch (err) {
      setError(toMessage(err));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="text-sm text-muted-foreground">
            {t(`kinds.${request.kind}`)}
            {request.request_no && (
              <span className="ms-2 font-mono text-xs" dir="ltr">
                {request.request_no}
              </span>
            )}
          </p>
          <h1 className="break-words text-2xl font-bold" dir="auto">
            {requestTitle(request)}
          </h1>
        </div>
        <div className="flex items-center gap-2">
          {canPrint && (
            <Button variant="outline" size="sm" render={<Link href={`/portal/requests/${request.id}/print`} />}>
              <Printer className="size-4" aria-hidden />
              {t("ui.printForm")}
            </Button>
          )}
          <StatusBadge status={request.status} />
        </div>
      </div>

      <Card>
        <CardContent className="pt-4">
          <RequestStepper kind={request.kind} status={request.status} events={detail.events} />
        </CardContent>
      </Card>

      {justSubmitted && <Alert variant="success">{t("ui.submitted")}</Alert>}
      {request.status === "draft" && isApplicant && (
        <Alert variant="pending">{t("ui.draftBanner")}</Alert>
      )}
      {request.kind === "research_plan" && detail.can_edit && (
        <Alert variant="info">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <span>{t("forms.planSignHint")}</span>
            <div className="flex flex-wrap gap-2">
              <Button variant="outline" size="sm" render={<Link href={`/portal/requests/${request.id}/print`} />}>
                <Printer className="size-4" aria-hidden />
                {t("forms.downloadFilled")}
              </Button>
              <OfficialFormButton kind={request.kind} />
            </div>
          </div>
        </Alert>
      )}
      {request.status === "returned" && isApplicant && (
        <Alert variant="warning">{t("ui.returnedBanner")}</Alert>
      )}
      {request.kind !== "research_plan" && (
        <RequestJournalPanel details={request.details} cached={detail.journal_flag} />
      )}

      {request.decision_note && (
        <div className="rounded-lg border-s-4 border-primary bg-muted p-3 text-sm">
          <p className="mb-1 text-xs text-muted-foreground">{t("ui.decisionNote")}</p>
          <p className="whitespace-pre-line">{request.decision_note}</p>
        </div>
      )}

      {letterIssued && (
        <div className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-status-success/30 bg-status-success-bg p-4 text-sm text-status-success-fg">
          <div className="flex items-center gap-2 font-semibold">
            <CheckCircle2 className="size-5 shrink-0" aria-hidden />
            {t("ui.letterIssuedBanner")}
          </div>
          <Button size="sm" render={<Link href={`/portal/letters/${detail.letter_id}`} />}>
            {t("ui.viewLetter")}
          </Button>
        </div>
      )}

      <div className="grid items-start gap-6 lg:grid-cols-3">
        <div className="space-y-6 lg:col-span-2">
          <Card>
            <CardHeader>
              <CardTitle>{t("ui.details")}</CardTitle>
            </CardHeader>
            <CardContent>
              <RequestSummary request={request} />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t("ui.attachments")}</CardTitle>
            </CardHeader>
            <CardContent>
              <AttachmentsPanel
                requestId={request.id}
                attachments={detail.attachments}
                uploadRoles={
                  detail.can_edit
                    ? APPLICANT_ATTACHMENT_ROLES[request.kind]
                    : reviewerMayChangeFiles
                      ? STAFF_ATTACHMENT_ROLES[request.kind]
                      : []
                }
                requiredRoles={required}
                canRemove={(file) =>
                  reviewerMayChangeFiles || (detail.can_edit && file.uploaded_by === user?.id)
                }
                onChanged={reload}
              />
            </CardContent>
          </Card>
        </div>

        <div className="space-y-6">
          {detail.can_edit && (
            <Card className="border-primary/30">
              <CardHeader>
                <CardTitle>{t("ui.step3")}</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                {error && <Alert variant="danger">{error}</Alert>}
                {missingRequired.length > 0 && (
                  <ul className="list-disc ps-5 text-sm text-status-pending-fg">
                    {missingRequired.map((role) => (
                      <li key={role}>
                        {t(`roles.${role}`)} — {t("ui.missing")}
                      </li>
                    ))}
                  </ul>
                )}
                <div className="flex flex-wrap gap-2">
                  <Button variant="outline" render={<Link href={`/portal/requests/${request.id}/edit`} />}>
                    {t("ui.edit")}
                  </Button>
                  <Button disabled={submitting || missingRequired.length > 0} onClick={submit}>
                    {t("ui.submit")}
                  </Button>
                </div>
              </CardContent>
            </Card>
          )}

          {canAct ? (
            <ReviewActions detail={detail} onChanged={reload} />
          ) : (
            !detail.can_edit && <WaitingNotice detail={detail} />
          )}

          <Card>
            <CardHeader>
              <CardTitle>{t("ui.history")}</CardTitle>
            </CardHeader>
            <CardContent>
              <Timeline events={detail.events} />
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}

/** Shows the cached result from the request detail until the live check answers. */
function RequestJournalPanel({ details, cached }: { details: PortalRequest["details"]; cached?: JournalCheck }) {
  const query = {
    issn: details.issn,
    issn_online: details.issn_online,
    name: details.journal_name,
    publisher: details.publisher,
    url: details.journal_url,
  };
  const { result, loading, failed, retry } = useJournalCheck(query, 0);
  if (!journalCheckParams(query) && !cached?.matched) return null;
  return <JournalEvidencePanel result={result ?? cached ?? null} loading={loading} failed={failed} onRetry={retry} />;
}

export default function RequestDetailPage() {
  return (
    <RequireAuth>
      <RequestPage />
    </RequireAuth>
  );
}
