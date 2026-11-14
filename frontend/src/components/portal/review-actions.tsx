"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { useRouter } from "@/i18n/routing";
import { api } from "@/lib/api";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import type { IssuedLetter, RequestDetail, RequestStatus } from "@/lib/portal/types";
import { NativeSelect, Textarea } from "./form-controls";
import { usePortalErrors } from "./use-portal-errors";

interface ReviewActionsProps {
  detail: RequestDetail;
  onChanged: () => void;
}

const DESTRUCTIVE: RequestStatus[] = ["rejected", "not_eligible"];

/** The outcomes scientific affairs may recommend, per request kind (mirrors the backend). */
function finalDecisions(kind: string): RequestStatus[] {
  return kind === "research_plan" || kind === "title_change"
    ? ["approved", "rejected"]
    : ["verified", "not_eligible"];
}

/**
 * Stage actions of the approval chain:
 * dean → initial approval / return · scientific affairs → recommendation to the VP ·
 * VP → confirm or send back · then the clerical steps (committee signatures, letter issue).
 */
export function ReviewActions({ detail, onChanged }: ReviewActionsProps) {
  const t = useTranslations("portal");
  const toMessage = usePortalErrors();
  const router = useRouter();
  const [note, setNote] = useState("");
  const [decision, setDecision] = useState("");
  const [diwanNo, setDiwanNo] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const { request, allowed_transitions: transitions } = detail;
  const canIssue = detail.can_issue_letter;
  const isDeanStage = request.status === "pending_dean" && transitions.length > 0;
  const isSAStage = transitions.includes("pending_vp" as RequestStatus);
  const isVPStage = request.status === "pending_vp" && transitions.length > 0;

  if (transitions.length === 0 && !canIssue) return null;

  const title = isDeanStage
    ? t("ui.deanActions")
    : isVPStage
      ? t("ui.vpActions")
      : t("ui.reviewActions");

  async function move(to: RequestStatus, withDecision?: string) {
    setError("");
    setBusy(true);
    try {
      await api.post(`/api/v1/portal/requests/${request.id}/transition`, {
        to,
        note,
        ...(withDecision ? { decision: withDecision } : {}),
      });
      setNote("");
      onChanged();
    } catch (err) {
      setError(toMessage(err));
    } finally {
      setBusy(false);
    }
  }

  async function issueLetter() {
    setError("");
    setBusy(true);
    try {
      const res = await api.post<IssuedLetter>(
        `/api/v1/portal/requests/${request.id}/issue-letter`,
        diwanNo.trim() ? { diwan_no: diwanNo.trim() } : {}
      );
      if (res.data) router.push(`/portal/letters/${res.data.id}`);
    } catch (err) {
      setError(toMessage(err));
      setBusy(false);
    }
  }

  function actionLabel(to: RequestStatus): string {
    if (request.status === "pending_dean" && to === "pending") return t("transitions.dean_approve");
    if (isVPStage && to === "pending") return t("transitions.back_to_sa");
    if (isVPStage) return t("transitions.vp_confirm", { decision: t(`statuses.${to}`) });
    return t(`transitions.${to}`);
  }

  return (
    <Card className="border-accent/30">
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {error && <Alert variant="danger">{error}</Alert>}

        {isVPStage && request.sa_decision && (
          <p className="rounded-lg bg-muted p-3 text-sm">
            {t("ui.saDecision")}: <b>{t(`statuses.${request.sa_decision}`)}</b>
          </p>
        )}

        {transitions.length > 0 && (
          <div className="space-y-2">
            <label htmlFor="review-note" className="text-sm font-medium">
              {t("ui.noteLabel")}
            </label>
            <Textarea id="review-note" value={note} onChange={(e) => setNote(e.target.value)} />
            {isVPStage && <p className="text-xs text-muted-foreground">{t("ui.noteVPHint")}</p>}
          </div>
        )}

        {isSAStage && (
          <div className="space-y-2 rounded-lg border p-3">
            <label htmlFor="sa-decision" className="text-sm font-medium">
              {t("ui.decisionLabel")}
            </label>
            <NativeSelect
              id="sa-decision"
              value={decision}
              onChange={(e) => setDecision(e.target.value)}
            >
              <option value="">{t("options.none")}</option>
              {finalDecisions(request.kind).map((d) => (
                <option key={d} value={d}>{t(`statuses.${d}`)}</option>
              ))}
            </NativeSelect>
            <Button
              type="button"
              disabled={busy || !decision}
              onClick={() => move("pending_vp" as RequestStatus, decision)}
            >
              {t("transitions.raise_to_vp")}
            </Button>
          </div>
        )}

        <div className="flex flex-wrap gap-2">
          {transitions
            .filter((to) => to !== "pending_vp")
            .map((to) => (
              <Button
                key={to}
                type="button"
                disabled={busy}
                variant={
                  DESTRUCTIVE.includes(to) ? "destructive"
                    : to === "returned" || (isVPStage && to === "pending") ? "outline"
                    : "default"
                }
                onClick={() => move(to)}
              >
                {actionLabel(to)}
              </Button>
            ))}
        </div>

        {canIssue && (
          <div className="space-y-2 rounded-lg border p-3">
            <label htmlFor="diwan-no" className="text-sm font-medium">
              {t("ui.diwanNoLabel")}
            </label>
            <Input
              id="diwan-no"
              dir="ltr"
              value={diwanNo}
              onChange={(e) => setDiwanNo(e.target.value)}
              placeholder={t("ui.diwanNoPlaceholder")}
            />
            <p className="text-xs text-muted-foreground">{t("ui.diwanNoHint")}</p>
            <Button type="button" disabled={busy} onClick={issueLetter}>
              {t("ui.issueLetter")}
            </Button>
            <p className="text-xs text-muted-foreground">{t("ui.issueLetterHint")}</p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
