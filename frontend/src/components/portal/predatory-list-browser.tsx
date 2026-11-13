"use client";

import { useEffect, useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import { Search } from "lucide-react";
import { api } from "@/lib/api";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { formatDate } from "@/lib/portal/types";
import { cn } from "@/lib/utils";

const LIST_TYPES = ["predatory_journal", "predatory_publisher", "hijacked"] as const;
type ListType = (typeof LIST_TYPES)[number];
const PAGE_SIZE = 50;

interface Entry {
  name: string;
  url: string | null;
  list_type: ListType;
  authentic_name: string | null;
  authentic_url: string | null;
  source: string;
  fetched_at: string;
}

function hostOf(url: string | null): string {
  if (!url) return "";
  try {
    return new URL(url).host.replace(/^www\./, "");
  } catch {
    return url;
  }
}

/** Searchable view of the predatory journals, publishers and hijacked-journal lists. */
export function PredatoryListBrowser() {
  const t = useTranslations("portal.predatory");
  const locale = useLocale();
  const [type, setType] = useState<ListType>("predatory_journal");
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  const [items, setItems] = useState<Entry[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    let active = true;
    const timer = setTimeout(() => {
      const params = new URLSearchParams({ type, page: String(page), limit: String(PAGE_SIZE) });
      if (query.trim()) params.set("q", query.trim());
      setLoading(true);
      api
        .get<Entry[]>(`/api/v1/screening/reference-list?${params}`)
        .then((res) => {
          if (!active) return;
          setItems(res.data ?? []);
          setTotal(res.meta?.total ?? 0);
          setFailed(false);
        })
        .catch(() => active && setFailed(true))
        .finally(() => active && setLoading(false));
    }, 250);
    return () => {
      active = false;
      clearTimeout(timer);
    };
  }, [type, query, page]);

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap gap-2" role="tablist" aria-label={t("listsTitle")}>
        {LIST_TYPES.map((lt) => (
          <Button
            key={lt}
            role="tab"
            aria-selected={type === lt}
            variant={type === lt ? "default" : "outline"}
            size="sm"
            onClick={() => {
              setType(lt);
              setPage(1);
            }}
          >
            {t(`lists.${lt}`)}
          </Button>
        ))}
      </div>
      <p className="text-sm text-muted-foreground">{t(`listHelp.${type}`)}</p>

      <div className="relative">
        <Search className="absolute start-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden />
        <Input
          className="ps-10"
          dir="auto"
          aria-label={t("searchPlaceholder")}
          placeholder={t("searchPlaceholder")}
          value={query}
          onChange={(e) => {
            setQuery(e.target.value);
            setPage(1);
          }}
        />
      </div>

      {failed && <Alert variant="danger">{t("loadFailed")}</Alert>}

      {loading ? (
        <div className="space-y-2">
          {[0, 1, 2, 3, 4].map((i) => (
            <Skeleton key={i} className="h-10 w-full" />
          ))}
        </div>
      ) : items.length === 0 ? (
        <p className="py-6 text-center text-sm text-muted-foreground">
          {query.trim() ? t("noMatch") : t("emptyList")}
        </p>
      ) : (
        <ul className="divide-y rounded-lg border">
          {items.map((e, i) => (
            <li key={`${e.name}-${i}`} className="space-y-1 p-3 text-sm">
              <p className="font-medium" dir="ltr">
                {e.name}
              </p>
              {e.url && (
                <p className="text-xs text-muted-foreground" dir="ltr">
                  {type === "hijacked" ? `${t("fakeSite")}: ` : ""}
                  <span className={cn(type === "hijacked" && "text-status-danger-fg")}>{hostOf(e.url)}</span>
                </p>
              )}
              {type === "hijacked" && (e.authentic_name || e.authentic_url) && (
                <p className="text-xs" dir="ltr">
                  {t("authentic")}:{" "}
                  {e.authentic_url ? (
                    <a
                      href={e.authentic_url}
                      target="_blank"
                      rel="noopener noreferrer nofollow"
                      className="text-status-success-fg hover:underline"
                    >
                      {e.authentic_name || hostOf(e.authentic_url)}
                    </a>
                  ) : (
                    e.authentic_name
                  )}
                </p>
              )}
            </li>
          ))}
        </ul>
      )}

      <div className="flex flex-wrap items-center justify-between gap-2 text-sm text-muted-foreground">
        <span className="tabular-nums">
          {t("count", { total })}
          {items[0] && ` · ${t("updatedOn", { date: formatDate(items[0].fetched_at, locale) })}`}
        </span>
        {total > PAGE_SIZE && (
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" disabled={page === 1} onClick={() => setPage((p) => p - 1)}>
              {t("previous")}
            </Button>
            <span className="tabular-nums">
              {page} / {Math.ceil(total / PAGE_SIZE)}
            </span>
            <Button
              variant="outline"
              size="sm"
              disabled={page * PAGE_SIZE >= total}
              onClick={() => setPage((p) => p + 1)}
            >
              {t("next")}
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
