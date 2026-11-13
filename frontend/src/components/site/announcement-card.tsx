"use client";

import { useTranslations } from "next-intl";
import { Paperclip } from "lucide-react";
import { Link } from "@/i18n/routing";
import { excerpt } from "@/lib/site/rich-text";
import type { PublicAnnouncement } from "@/lib/site/types";
import { cn } from "@/lib/utils";
import { AnnouncementMeta } from "./announcement-meta";

const EXCERPT_LENGTH = 180;

/** One announcement in a list; the whole card links to its detail page. */
export function AnnouncementCard({ item, compact = false }: { item: PublicAnnouncement; compact?: boolean }) {
  const t = useTranslations("site.news");
  const summary = excerpt(item.body, EXCERPT_LENGTH);
  return (
    <article
      className={cn(
        "group relative rounded-xl border bg-card p-4 transition-shadow hover:shadow-md focus-within:ring-3 focus-within:ring-ring/40",
        item.pinned && "border-s-4 border-s-accent"
      )}
    >
      <AnnouncementMeta item={item} />
      <h3 className={cn("mt-2 font-bold text-primary", compact ? "text-base" : "text-lg")}>
        <Link href={`/news/${item.id}`} className="outline-none after:absolute after:inset-0 group-hover:text-accent">
          {item.title}
        </Link>
      </h3>
      {summary && !compact && <p className="mt-2 text-sm leading-relaxed text-muted-foreground">{summary}</p>}
      {item.attachment && (
        <p className="mt-2 inline-flex items-center gap-1 text-xs text-muted-foreground">
          <Paperclip className="size-3.5" aria-hidden />
          {t("attachment")}
        </p>
      )}
    </article>
  );
}
