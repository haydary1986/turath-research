"use client";

import { useLocale, useTranslations } from "next-intl";
import { StatusBadge } from "./status-badge";
import { formatDateTime, type RequestEvent, type RequestStatus } from "@/lib/portal/types";
import { Bidi, Ltr } from "@/components/i18n/bidi";
import { Badge } from "@/components/ui/badge";
import { FAMILY_TINT } from "@/lib/portal/status";

/** History entry the backend adds to a plan when an approved title-change request renames it. */
const TITLE_CHANGED = "title_changed";

export function Timeline({ events }: { events: RequestEvent[] }) {
  const t = useTranslations("portal.ui");
  const locale = useLocale();

  return (
    <ol className="space-y-4 border-s ps-4">
      {events.map((event, index) => (
        <li key={`${event.created_at}-${index}`} className="relative">
          <span className="absolute -start-[1.3rem] top-1.5 size-2.5 rounded-full bg-primary" />
          <div className="flex flex-wrap items-center gap-2">
            {event.to_status === TITLE_CHANGED ? (
              <Badge variant="secondary" className={FAMILY_TINT.info}>{t("titleChanged")}</Badge>
            ) : (
              <StatusBadge status={event.to_status as RequestStatus} />
            )}
            <Ltr className="text-xs text-muted-foreground">{formatDateTime(event.created_at, locale)}</Ltr>
          </div>
          {event.actor_name && (
            <p className="mt-1 text-xs text-muted-foreground">
              {t("by")} <Bidi text={event.actor_name} />
            </p>
          )}
          <Bidi as="p" block text={event.note} className="mt-1 whitespace-pre-line text-sm" />
        </li>
      ))}
    </ol>
  );
}
