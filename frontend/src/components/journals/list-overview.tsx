import { getTranslations } from "next-intl/server";
import { BookX, Building2, Link2Off, type LucideIcon } from "lucide-react";
import { Link } from "@/i18n/routing";
import { JOURNAL_PATHS, LIST_TYPES, type JournalListStatus, type ListType } from "@/lib/journals/lists";
import { formatDate } from "@/lib/portal/types";

const LIST_ICONS: Record<ListType, LucideIcon> = {
  predatory_journal: BookX,
  predatory_publisher: Building2,
  hijacked: Link2Off,
};

/** The three reference lists with their sizes and refresh dates, each linking to its A–Z index. */
export async function ListOverview({ locale, statuses }: { locale: string; statuses: JournalListStatus[] }) {
  const t = await getTranslations({ locale, namespace: "journals" });
  return (
    <ul className="grid gap-3 md:grid-cols-3">
      {LIST_TYPES.map((type) => {
        const status = statuses.find((s) => s.list_type === type);
        const Icon = LIST_ICONS[type];
        return (
          <li key={type}>
            <Link
              href={JOURNAL_PATHS.index(type)}
              className="group flex h-full flex-col rounded-xl border bg-card p-4 hover:border-accent/40 hover:shadow-sm focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
            >
              <span className="flex items-center gap-2 font-bold text-primary group-hover:text-accent">
                <Icon className="size-5 text-accent" aria-hidden />
                {t(`lists.${type}`)}
              </span>
              <span className="mt-2 text-sm text-muted-foreground">{t(`listHelp.${type}`)}</span>
              {status && (
                <span className="mt-auto pt-3 text-xs text-muted-foreground">
                  <span className="font-semibold text-foreground tabular-nums">{t("listPage.count", { count: status.count })}</span>
                  {status.last_success_at && <> · {t("listPage.lastRefresh", { date: formatDate(status.last_success_at, locale) })}</>}
                </span>
              )}
            </Link>
          </li>
        );
      })}
    </ul>
  );
}
