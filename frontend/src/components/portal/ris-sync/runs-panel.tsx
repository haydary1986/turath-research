"use client";

import { useLocale, useTranslations } from "next-intl";
import { FlaskConical, RefreshCw } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Ltr } from "@/components/i18n/bidi";
import { formatDateTime } from "@/lib/portal/types";
import type { RISSyncRun, RISSyncStatus } from "@/lib/portal/ris-sync";

interface RunsPanelProps {
  status: RISSyncStatus;
  busy: boolean;
  selectedRun: string | null;
  onStart: (mode: "dry_run" | "apply") => void;
  onSelect: (run: RISSyncRun) => void;
}

const COUNTS = ["matched", "to_ris", "to_portal", "reviews", "skipped", "failed"] as const;

/** Configuration, the run buttons and the latest runs with their counts. */
export function RunsPanel({ status, busy, selectedRun, onStart, onSelect }: RunsPanelProps) {
  const t = useTranslations("portal.risSync");
  const locale = useLocale();
  const running = status.runs.some((r) => !r.finished_at);

  return (
    <Card>
      <CardHeader className="space-y-3">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <CardTitle>{t("runsTitle")}</CardTitle>
          <div className="flex flex-wrap gap-2">
            <Button variant="outline" disabled={busy || running || !status.configured} onClick={() => onStart("dry_run")}>
              <FlaskConical className="size-4" aria-hidden />
              {t("dryRun")}
            </Button>
            <Button disabled={busy || running || !status.configured} onClick={() => onStart("apply")}>
              <RefreshCw className={`size-4 ${running ? "animate-spin" : ""}`} aria-hidden />
              {running ? t("running") : t("apply")}
            </Button>
          </div>
        </div>
        <div className="flex flex-wrap gap-2 text-sm">
          <Badge variant="secondary">{status.configured ? t("configured") : t("notConfigured")}</Badge>
          <Badge variant="secondary">
            {status.scheduled ? t("scheduled", { minutes: status.interval_minutes }) : t("manualOnly")}
          </Badge>
          <Badge variant="secondary">{t("linked", { count: status.linked_profiles })}</Badge>
          <Badge variant="secondary">{t("openReviews", { count: status.open_reviews })}</Badge>
        </div>
        <p className="text-xs text-muted-foreground">{t("hint")}</p>
      </CardHeader>
      <CardContent className="overflow-x-auto">
        {status.runs.length === 0 ? (
          <p className="py-6 text-center text-sm text-muted-foreground">{t("noRuns")}</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("startedAt")}</TableHead>
                <TableHead>{t("mode")}</TableHead>
                {COUNTS.map((k) => <TableHead key={k} className="text-end">{t(`counts.${k}`)}</TableHead>)}
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {status.runs.map((run) => (
                <TableRow key={run.id} data-state={selectedRun === run.id ? "selected" : undefined}>
                  <TableCell><Ltr>{formatDateTime(run.started_at, locale)}</Ltr></TableCell>
                  <TableCell>
                    <Badge variant="outline">{t(`modes.${run.mode}`)}</Badge>
                    {!run.finished_at && <span className="ms-2 text-xs text-muted-foreground">{t("running")}</span>}
                    {run.failed && <span className="ms-2 text-xs text-status-danger-fg">{t("runFailed")}</span>}
                  </TableCell>
                  {COUNTS.map((k) => (
                    <TableCell key={k} className="text-end tabular-nums">{run.stats[k] ?? "—"}</TableCell>
                  ))}
                  <TableCell>
                    <Button variant="ghost" size="sm" disabled={!run.finished_at} onClick={() => onSelect(run)}>
                      {t("details")}
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}
