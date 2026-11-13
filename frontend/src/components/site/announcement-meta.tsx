"use client";

import { useLocale, useTranslations } from "next-intl";
import { CalendarDays, Hash, Pin } from "lucide-react";
import { FAMILY_TINT } from "@/lib/portal/status";
import { formatDate } from "@/lib/portal/types";
import type { AnnouncementType, PublicAnnouncement } from "@/lib/site/types";
import { cn } from "@/lib/utils";

const TYPE_TINT: Record<AnnouncementType, string> = {
  circular: FAMILY_TINT.danger,
  activity: FAMILY_TINT.info,
};

export function AnnouncementTypeBadge({ type }: { type: AnnouncementType }) {
  const t = useTranslations("site.types");
  return (
    <span className={cn("inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold", TYPE_TINT[type])}>
      {t(type)}
    </span>
  );
}

/** Type, pin, circular number, event date and publication date of an announcement. */
export function AnnouncementMeta({ item, className }: { item: PublicAnnouncement; className?: string }) {
  const t = useTranslations("site.news");
  const locale = useLocale();
  return (
    <div className={cn("flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground", className)}>
      <AnnouncementTypeBadge type={item.type} />
      {item.pinned && (
        <span className="inline-flex items-center gap-1 font-medium text-accent">
          <Pin className="size-3.5" aria-hidden />
          {t("pinned")}
        </span>
      )}
      {item.reference_no && (
        <span className="inline-flex items-center gap-1">
          <Hash className="size-3.5" aria-hidden />
          {t("circularNo", { no: item.reference_no })}
        </span>
      )}
      {item.event_date && (
        <span className="inline-flex items-center gap-1">
          <CalendarDays className="size-3.5" aria-hidden />
          {t("eventDate", { date: formatDate(item.event_date, locale) })}
        </span>
      )}
      {item.published_at && <span>{t("publishedOn", { date: formatDate(item.published_at, locale) })}</span>}
    </div>
  );
}
