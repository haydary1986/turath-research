"use client";

import { useCallback, useEffect, useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import { api } from "@/lib/api";
import { BULK_SYNC_ENDPOINT, type BulkSyncProgress } from "@/lib/portal/research-profile";
import { usePortalErrors } from "@/components/portal/use-portal-errors";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

const POLL_INTERVAL_MS = 5000;

/** Admin control for the sequential ORCID sync: start, stop, resume and progress counts. */
export function BulkSyncPanel() {
  const t = useTranslations("portal.bulkSync");
  const syncErrors = useTranslations("portal.profile.syncErrors");
  const locale = useLocale();
  const toMessage = usePortalErrors();
  const [progress, setProgress] = useState<BulkSyncProgress | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const load = useCallback(() => {
    api
      .get<BulkSyncProgress>(BULK_SYNC_ENDPOINT)
      .then((res) => setProgress(res.data ?? null))
      .catch(() => setError(t("loadError")));
  }, [t]);

  useEffect(load, [load]);

  const active = progress?.active ?? false;
  useEffect(() => {
    if (!active) return;
    const timer = setInterval(load, POLL_INTERVAL_MS);
    return () => clearInterval(timer);
  }, [active, load]);

  const post = (path: string, body: object) => {
    setBusy(true);
    setError("");
    api
      .post<BulkSyncProgress>(`${BULK_SYNC_ENDPOINT}${path}`, body)
      .then((res) => setProgress(res.data ?? null))
      .catch((err) => setError(toMessage(err)))
      .finally(() => setBusy(false));
  };

  if (!progress) return error ? <Alert variant="danger">{error}</Alert> : null;

  const run = progress.run;
  const counts = [
    { label: t("eligible"), value: progress.eligible },
    ...(run
      ? [
          { label: t("processed"), value: run.processed },
          { label: t("remaining"), value: progress.remaining },
          { label: t("succeeded"), value: run.succeeded },
          { label: t("failed"), value: run.failed },
          { label: t("skipped"), value: run.skipped },
        ]
      : []),
  ];
  const stopped = run?.status === "stopped";

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("title")}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-sm text-muted-foreground">{t("intro")}</p>
        {error && <Alert variant="danger">{error}</Alert>}
        {run && <p className="text-sm font-medium">{t(`status.${run.status}`)}</p>}
        {run?.wait_reason && run.resume_at && t.has(`wait.${run.wait_reason}`) && (
          <Alert variant="warning">
            {t(`wait.${run.wait_reason}`, { time: new Date(run.resume_at).toLocaleString(locale) })}
          </Alert>
        )}
        <div className="flex flex-wrap gap-4 text-sm">
          {counts.map(({ label, value }) => (
            <span key={label}>
              {label}: <b className="tabular-nums">{value}</b>
            </span>
          ))}
        </div>
        {run && Object.keys(run.failures).length > 0 && (
          <div className="text-sm">
            <p className="font-medium">{t("failuresTitle")}</p>
            <ul className="list-disc ps-5 text-muted-foreground">
              {Object.entries(run.failures).map(([code, count]) => (
                <li key={code}>
                  {syncErrors.has(code) ? syncErrors(code) : syncErrors("generic")}: {count}
                </li>
              ))}
            </ul>
          </div>
        )}
        <div className="flex flex-wrap gap-2">
          {active ? (
            <Button variant="outline" disabled={busy} onClick={() => post("/stop", {})}>
              {t("stop")}
            </Button>
          ) : (
            <>
              <Button disabled={busy || progress.eligible === 0} onClick={() => post("", {})}>
                {stopped ? t("resume") : run ? t("restart") : t("start")}
              </Button>
              {stopped && (
                <Button variant="outline" disabled={busy} onClick={() => post("", { fresh: true })}>
                  {t("restart")}
                </Button>
              )}
              {run && (
                <Button variant="outline" disabled={busy} onClick={() => post("", { only_failed: true, fresh: true })}>
                  {t("retryFailed")}
                </Button>
              )}
            </>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
