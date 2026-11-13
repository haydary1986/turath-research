"use client";

import { useLocale, useTranslations } from "next-intl";
import { CircleHelp, ExternalLink, Loader2, ShieldAlert, ShieldX, SearchCheck } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { FAMILY_TINT, JOURNAL_VERDICT_FAMILY, type StatusFamily } from "@/lib/portal/status";
import type { JournalCheck, JournalSourceEvidence, JournalSourceStatus } from "@/lib/portal/types";
import { JournalFlagBanner } from "./journal-flag-banner";

const VERDICT_ICON: Record<JournalCheck["verdict"], LucideIcon> = {
  danger: ShieldX,
  caution: ShieldAlert,
  found: SearchCheck,
  unknown: CircleHelp,
};

const SOURCE_STATUS_FAMILY: Record<JournalSourceStatus, StatusFamily> = {
  found: "info",
  not_found: "neutral",
  failed: "warning",
  not_checked: "neutral",
  unknown: "neutral",
};

const SOURCE_KEYS = ["openalex", "crossref", "doaj"] as const;

function SourceRow({ name, evidence }: { name: string; evidence: JournalSourceEvidence }) {
  const t = useTranslations("portal.journals.check");
  const facts: string[] = [];
  if (evidence.type) facts.push(t("facts.type", { value: evidence.type }));
  if (evidence.h_index != null) facts.push(t("facts.hIndex", { value: evidence.h_index }));
  if (evidence.two_year_mean_citedness != null) {
    facts.push(t("facts.citedness", { value: evidence.two_year_mean_citedness.toFixed(2) }));
  }
  if (evidence.works_count != null) facts.push(t("facts.works", { value: evidence.works_count }));
  if (evidence.total_dois != null) facts.push(t("facts.dois", { value: evidence.total_dois }));
  if (evidence.in_doaj && name !== t("sources.doaj")) facts.push(t("facts.inDoaj"));

  return (
    <li className="flex flex-wrap items-start justify-between gap-2 border-b py-2 last:border-b-0">
      <div className="min-w-0">
        <p className="font-medium" dir="ltr">{name}</p>
        {evidence.title && <p className="text-xs" dir="auto">{evidence.title}</p>}
        {facts.length > 0 && <p className="text-xs text-muted-foreground">{facts.join(" · ")}</p>}
        {evidence.url && (
          <a href={evidence.url} target="_blank" rel="noopener noreferrer nofollow"
            className="inline-flex items-center gap-1 text-xs text-primary hover:underline">
            <ExternalLink className="size-3" aria-hidden />
            {t("facts.open")}
          </a>
        )}
      </div>
      <Badge variant="secondary" className={FAMILY_TINT[SOURCE_STATUS_FAMILY[evidence.status]]}>
        {t(`status.${evidence.status}`)}
      </Badge>
    </li>
  );
}

interface PanelProps {
  result: JournalCheck | null;
  loading?: boolean;
  failed?: boolean;
  onRetry?: () => void;
  children?: React.ReactNode;
}

/** Advisory evidence from the automatic check, shared by the form, request detail and journals page. */
export function JournalEvidencePanel({ result, loading, failed, onRetry, children }: PanelProps) {
  const t = useTranslations("portal.journals.check");
  const locale = useLocale();
  if (!result && !loading && !failed) return null;

  const family = result ? JOURNAL_VERDICT_FAMILY[result.verdict] : "neutral";
  const Icon = result ? VERDICT_ICON[result.verdict] : CircleHelp;
  const separator = locale === "ar" ? "، " : ", ";

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex flex-wrap items-center justify-between gap-2 text-base">
          <span>{t("panelTitle")}</span>
          {loading && <Loader2 className="size-4 animate-spin text-muted-foreground" aria-label={t("checking")} />}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3 text-sm">
        {failed && (
          <div className="space-y-2">
            <Alert variant="warning">{t("failed")}</Alert>
            {onRetry && <Button type="button" variant="outline" size="sm" onClick={onRetry}>{t("retry")}</Button>}
          </div>
        )}
        {result && (
          <>
            <div className={`flex items-start gap-2 rounded-lg p-3 font-semibold ${FAMILY_TINT[family]}`}>
              <Icon className="mt-0.5 size-5 shrink-0" aria-hidden />
              <span>{t(`verdict.${result.verdict}`)}</span>
            </div>
            <p className="text-xs text-muted-foreground">{t("advisory")}</p>

            <JournalFlagBanner flag={result} />

            {result.reasons.length > 0 && (
              <ul className="list-disc space-y-1 ps-5">
                {result.reasons.map((reason) => (
                  <li key={reason}>{t.has(`reasons.${reason}`) ? t(`reasons.${reason}`) : reason}</li>
                ))}
              </ul>
            )}

            {result.throttled && <Alert variant="warning">{t("throttled")}</Alert>}

            {result.sources_failed.length > 0 && (
              <Alert variant="warning">
                {t("sourcesFailed", {
                  sources: result.sources_failed
                    .map((source) => (t.has(`sources.${source}`) ? t(`sources.${source}`) : source))
                    .join(separator),
                })}
              </Alert>
            )}

            {result.reference_hits.length > 0 && (
              <div className="space-y-2">
                <p className="font-medium">{t("listHitsTitle")}</p>
                <ul className="space-y-2">
                  {result.reference_hits.map((hit) => (
                    <li key={`${hit.list}-${hit.name}-${hit.matched_by}`} className="rounded-lg border p-2">
                      <p className="font-medium">{t(`lists.${hit.list}`)} · {t(`matchedBy.${hit.matched_by}`)}</p>
                      <p dir="auto">{hit.name}</p>
                      {hit.url && <a href={hit.url} target="_blank" rel="noopener noreferrer nofollow" dir="ltr" className="block break-all text-xs text-muted-foreground">{hit.url}</a>}
                      {hit.authentic_name && <p className="text-xs">{t("authentic", { name: hit.authentic_name })}</p>}
                      {hit.authentic_url && <a href={hit.authentic_url} target="_blank" rel="noopener noreferrer nofollow" dir="ltr" className="block break-all text-xs text-primary hover:underline">{hit.authentic_url}</a>}
                    </li>
                  ))}
                </ul>
                <p className="text-xs text-muted-foreground">{t("listCaveat")}</p>
              </div>
            )}

            {(result.canonical_name || result.publisher || result.issns.length > 0) && (
              <dl className="grid gap-1 sm:grid-cols-[auto_1fr] sm:gap-x-3">
                {result.canonical_name && (<><dt className="text-muted-foreground">{t("canonicalName")}</dt><dd dir="auto">{result.canonical_name}</dd></>)}
                {result.publisher && (<><dt className="text-muted-foreground">{t("publisher")}</dt><dd dir="auto">{result.publisher}</dd></>)}
                {result.issns.length > 0 && (<><dt className="text-muted-foreground">{t("issns")}</dt><dd dir="ltr" className="text-start">{result.issns.join(", ")}</dd></>)}
              </dl>
            )}

            {children}

            <div>
              <p className="font-medium">{t("sourcesTitle")}</p>
              <ul>
                {SOURCE_KEYS.map((key) => (
                  <SourceRow key={key} name={t(`sources.${key}`)} evidence={result.evidence[key]} />
                ))}
                <SourceRow name={t("sources.scopus")} evidence={result.evidence.scopus} />
              </ul>
              <p className="mt-1 text-xs text-muted-foreground">{t("scopusNote")}</p>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}

