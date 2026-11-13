"use client";

import { useEffect, useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import { Search } from "lucide-react";
import { api } from "@/lib/api";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  MIN_NAME_SEARCH_LENGTH, journalCheckParams, parseJournalSearch, type JournalQuery,
} from "@/lib/portal/journal-query";
import type { JournalListStatus } from "@/lib/portal/types";
import { JournalEvidencePanel } from "./journal-evidence-panel";
import { useJournalCheck } from "./use-journal-check";

function ListStatusCard() {
  const t = useTranslations("portal.journals.check");
  const locale = useLocale();
  const [statuses, setStatuses] = useState<JournalListStatus[] | null>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    let active = true;
    api
      .get<JournalListStatus[]>("/api/v1/screening/status")
      .then((res) => active && setStatuses(res.data ?? []))
      .catch(() => active && setFailed(true));
    return () => {
      active = false;
    };
  }, []);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{t("listStatus.title")}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-2 text-sm">
        {failed && <Alert variant="warning">{t("failed")}</Alert>}
        <ul className="grid gap-3 md:grid-cols-3">
          {statuses?.map((status) => (
            <li key={status.list_type} className="rounded-lg border p-3">
              <p className="font-medium">{t(`lists.${status.list_type}`)}</p>
              <p>{t("listStatus.count", { count: status.count })}</p>
              <p className="text-xs text-muted-foreground">
                {status.last_success_at
                  ? t("listStatus.lastRefresh", { date: new Date(status.last_success_at).toLocaleString(locale) })
                  : t("listStatus.never")}
              </p>
              {status.last_failed && <p className="text-xs text-status-warning-fg">{t("listStatus.failed")}</p>}
            </li>
          ))}
        </ul>
        <p className="text-xs text-muted-foreground">{t("listStatus.source")}</p>
      </CardContent>
    </Card>
  );
}

/** The "check a journal" search, open to every signed-in role. */
export function JournalCheckSearch() {
  const t = useTranslations("portal.journals.check");
  const [text, setText] = useState("");
  const [query, setQuery] = useState<JournalQuery>({});
  const [tooShort, setTooShort] = useState(false);
  const { result, loading, failed, retry } = useJournalCheck(query, 0);

  function submit(e: React.FormEvent) {
    e.preventDefault();
    const parsed = parseJournalSearch(text);
    setTooShort(parsed === null);
    const next = parsed ?? {};
    // The same query changes no state, so asking again after a failure must retry explicitly.
    if (failed && journalCheckParams(next) === journalCheckParams(query)) retry();
    setQuery(next);
  }

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>{t("title")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          <form onSubmit={submit} className="flex flex-wrap gap-2">
            <div className="relative min-w-56 flex-1">
              <Search className="absolute start-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden />
              <Input className="ps-10" dir="auto" aria-label={t("title")} placeholder={t("placeholder")}
                aria-invalid={tooShort} aria-describedby={tooShort ? "journal-search-short" : undefined}
                value={text} onChange={(e) => { setText(e.target.value); setTooShort(false); }} />
            </div>
            <Button type="submit" disabled={loading}>{loading ? t("checking") : t("submit")}</Button>
          </form>
          {tooShort && (
            <div id="journal-search-short">
              <Alert variant="warning">{t("tooShort", { min: MIN_NAME_SEARCH_LENGTH })}</Alert>
            </div>
          )}
          <p className="text-xs text-muted-foreground">{t("hint")}</p>
        </CardContent>
      </Card>
      <JournalEvidencePanel result={result} loading={loading} failed={failed} onRetry={retry} />
      <ListStatusCard />
    </div>
  );
}
