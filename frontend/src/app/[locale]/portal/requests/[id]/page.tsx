"use client";

import { useParams } from "next/navigation";
import { useState } from "react";
import { useTranslations } from "next-intl";
import { Link } from "@/i18n/routing";
import { api } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { AttachmentsPanel } from "@/components/portal/attachments-panel";
import { JournalFlagBanner } from "@/components/portal/journal-flag-banner";
import { RequireAuth } from "@/components/portal/require-auth";
import { RequestSummary } from "@/components/portal/request-summary";
import { ReviewActions } from "@/components/portal/review-actions";
import { StatusBadge } from "@/components/portal/status-badge";
import { Timeline } from "@/components/portal/timeline";
import { usePortalErrors } from "@/components/portal/use-portal-errors";
import { useRequestDetail } from "@/components/portal/use-request-detail";
import { requiredRoles } from "@/lib/portal/fields";
import {
  APPLICANT_ATTACHMENT_ROLES,
  STAFF_ATTACHMENT_ROLES,
  requestTitle,
} from "@/lib/portal/types";
import { useAuth } from "@/lib/auth";
import { isStaffRole } from "@/lib/portal/types";

function RequestPage() {
  const params = useParams<{ id: string }>();
  const t = useTranslations("portal");
  const toMessage = usePortalErrors();
  const { user } = useAuth();
  const { detail, loading, notFound, reload } = useRequestDetail(params.id);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  if (loading) return <p className="text-center text-muted-foreground">{t("ui.loading")}</p>;
  if (notFound || !detail) {
    return <p className="text-center text-muted-foreground">{t("ui.notFound")}</p>;
  }

  const { request } = detail;
  const isApplicant = user?.id === request.applicant_id;
  const isReviewer = isStaffRole(user?.role) && !isApplicant;
  const required = requiredRoles(request.kind, !!request.related_request_id);
  const present = new Set(detail.attachments.map((a) => a.role));
  const missingRequired = required.filter((role) => !present.has(role));

  async function submit() {
    setError("");
    setSubmitting(true);
    try {
      await api.post(`/api/v1/portal/requests/${request.id}/submit`, {});
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
          <p className="text-sm text-muted-foreground">{t(`kinds.${request.kind}`)}</p>
          <h1 className="break-words text-2xl font-bold" dir="auto">
            {requestTitle(request)}
          </h1>
        </div>
        <StatusBadge status={request.status} />
      </div>

      {request.status === "draft" && isApplicant && (
        <div className="rounded bg-amber-50 p-3 text-sm text-amber-800">{t("ui.draftBanner")}</div>
      )}
      {request.status === "returned" && isApplicant && (
        <div className="rounded bg-orange-50 p-3 text-sm text-orange-800">{t("ui.returnedBanner")}</div>
      )}
      <JournalFlagBanner flag={detail.journal_flag} />

      {request.decision_note && (
        <div className="rounded border-s-4 border-primary bg-muted p-3 text-sm">
          <p className="mb-1 text-xs text-muted-foreground">{t("ui.decisionNote")}</p>
          <p className="whitespace-pre-line">{request.decision_note}</p>
        </div>
      )}

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
                : isReviewer && request.status !== "draft"
                  ? STAFF_ATTACHMENT_ROLES[request.kind]
                  : []
            }
            requiredRoles={required}
            canRemove={detail.can_edit || isReviewer}
            onChanged={reload}
          />
        </CardContent>
      </Card>

      {detail.can_edit && (
        <Card>
          <CardHeader>
            <CardTitle>{t("ui.step3")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {error && (
              <div className="whitespace-pre-line rounded bg-red-50 p-3 text-sm text-red-600">{error}</div>
            )}
            {missingRequired.length > 0 && (
              <ul className="list-disc ps-5 text-sm text-amber-700">
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
              <Button disabled={submitting} onClick={submit}>
                {t("ui.submit")}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {isReviewer && <ReviewActions detail={detail} onChanged={reload} />}

      {isApplicant && detail.letter_id && (
        <Button render={<Link href={`/portal/letters/${detail.letter_id}`} />}>{t("ui.viewLetter")}</Button>
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
  );
}

export default function RequestDetailPage() {
  return (
    <RequireAuth>
      <RequestPage />
    </RequireAuth>
  );
}
