"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { useRouter } from "@/i18n/routing";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { PUBLIC_API, type AnnouncementType, type PublicAnnouncement } from "@/lib/site/types";
import { cn } from "@/lib/utils";
import { AnnouncementCard } from "./announcement-card";
import { usePublicResource } from "./use-public-resource";

const PAGE_SIZE = 10;
const FILTERS: (AnnouncementType | "all")[] = ["all", "circular", "activity"];

interface NewsListProps {
  initialType: AnnouncementType | null;
  initialPage: number;
}

function queryString(type: AnnouncementType | null, page: number): string {
  const params = new URLSearchParams();
  if (type) params.set("type", type);
  if (page > 1) params.set("page", String(page));
  const s = params.toString();
  return s ? `?${s}` : "";
}

/** Published circulars and activities with a type filter and pagination mirrored in the URL. */
export function NewsList({ initialType, initialPage }: NewsListProps) {
  const t = useTranslations("site");
  const router = useRouter();
  const [type, setType] = useState<AnnouncementType | null>(initialType);
  const [page, setPage] = useState(initialPage);

  const params = new URLSearchParams({ page: String(page), limit: String(PAGE_SIZE) });
  if (type) params.set("type", type);
  const { data, meta, error, loading, reload } = usePublicResource<PublicAnnouncement[]>(
    `${PUBLIC_API.announcements}?${params}`
  );
  const pages = Math.max(1, Math.ceil((meta?.total ?? 0) / PAGE_SIZE));

  function go(nextType: AnnouncementType | null, nextPage: number) {
    setType(nextType);
    setPage(nextPage);
    router.replace(`/news${queryString(nextType, nextPage)}`, { scroll: false });
  }

  return (
    <div className="space-y-6">
      <div role="group" aria-label={t("news.filterLabel")} className="flex flex-wrap gap-2">
        {FILTERS.map((f) => {
          const value = f === "all" ? null : f;
          const active = value === type;
          return (
            <button
              key={f}
              type="button"
              aria-pressed={active}
              onClick={() => go(value, 1)}
              className={cn(
                "rounded-full border px-4 py-1.5 text-sm font-medium transition-colors focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none",
                active ? "border-primary bg-primary text-primary-foreground" : "bg-card text-foreground hover:border-accent hover:text-accent"
              )}
            >
              {t(`news.${f}`)}
            </button>
          );
        })}
      </div>

      {loading && (
        <div className="space-y-3" aria-hidden>
          {[0, 1, 2, 3].map((i) => <Skeleton key={i} className="h-28 w-full" />)}
        </div>
      )}
      {error && (
        <Alert variant="warning">
          {t("home.loadError")}{" "}
          <button type="button" onClick={reload} className="font-semibold underline">{t("home.retry")}</button>
        </Alert>
      )}
      {!loading && !error && (data?.length ? (
        <div className="space-y-3">
          {data.map((item) => <AnnouncementCard key={item.id} item={item} />)}
        </div>
      ) : (
        <p className="rounded-xl border border-dashed p-8 text-center text-muted-foreground">{t("news.empty")}</p>
      ))}

      {pages > 1 && (
        <nav className="flex items-center justify-center gap-3" aria-label={t("news.pageOf", { page, pages })}>
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => go(type, page - 1)}>
            {t("news.previous")}
          </Button>
          <span className="text-sm text-muted-foreground" aria-live="polite">{t("news.pageOf", { page, pages })}</span>
          <Button variant="outline" size="sm" disabled={page >= pages} onClick={() => go(type, page + 1)}>
            {t("news.next")}
          </Button>
        </nav>
      )}
    </div>
  );
}
