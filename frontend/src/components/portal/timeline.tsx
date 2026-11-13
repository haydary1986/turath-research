"use client";

import { useLocale, useTranslations } from "next-intl";
import { StatusBadge } from "./status-badge";
import { formatDateTime, type RequestEvent, type RequestStatus } from "@/lib/portal/types";

export function Timeline({ events }: { events: RequestEvent[] }) {
  const t = useTranslations("portal.ui");
  const locale = useLocale();

  return (
    <ol className="space-y-4 border-s ps-4">
      {events.map((event, index) => (
        <li key={`${event.created_at}-${index}`} className="relative">
          <span className="absolute -start-[1.3rem] top-1.5 size-2.5 rounded-full bg-primary" />
          <div className="flex flex-wrap items-center gap-2">
            <StatusBadge status={event.to_status as RequestStatus} />
            <span className="text-xs text-muted-foreground">
              {formatDateTime(event.created_at, locale)}
            </span>
          </div>
          {event.actor_name && (
            <p className="mt-1 text-xs text-muted-foreground">
              {t("by")} {event.actor_name}
            </p>
          )}
          {event.note && <p className="mt-1 whitespace-pre-line text-sm">{event.note}</p>}
        </li>
      ))}
    </ol>
  );
}
