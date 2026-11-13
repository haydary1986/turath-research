import { getTranslations } from "next-intl/server";
import { BookOpenCheck, CalendarCheck, GraduationCap, Landmark, Medal, type LucideIcon } from "lucide-react";
import { formatNumber } from "@/lib/site/format";
import type { PublicStat, StatKey } from "@/lib/site/types";
import { cn } from "@/lib/utils";

const STAT_ICONS: Record<StatKey, LucideIcon> = {
  academics: GraduationCap,
  publications: BookOpenCheck,
  indexed: Medal,
  colleges: Landmark,
  activities: CalendarCheck,
};

/** Formats a statistic for display: the manual value as typed, or the computed number. */
export function formatStat(stat: Pick<PublicStat, "value" | "number">, locale: string): string {
  if (stat.number === null) return stat.value;
  return formatNumber(stat.number, locale);
}

interface StatsBarProps {
  locale: string;
  stats: PublicStat[];
  className?: string;
}

/** The live statistics under the hero; hidden when every statistic is hidden. */
export async function StatsBar({ locale, stats, className }: StatsBarProps) {
  if (stats.length === 0) return null;
  const t = await getTranslations({ locale, namespace: "landing.stats" });
  return (
    <section aria-labelledby="stats-title" className={cn("relative z-10 mx-auto -mt-8 max-w-6xl px-4", className)}>
      <h2 id="stats-title" className="sr-only">{t("title")}</h2>
      {/* Wrapping flex rows: cards grow to fill each row, so no row has an empty cell. */}
      <dl className="flex flex-wrap gap-px overflow-hidden rounded-2xl border bg-border shadow-lg">
        {stats.map((stat) => {
          const Icon = STAT_ICONS[stat.key] ?? Medal;
          return (
            <div
              key={stat.key}
              className="flex min-w-[calc(50%-1px)] flex-1 items-center gap-3 bg-card px-4 py-5 sm:min-w-[calc(33.333%-1px)] lg:min-w-0"
            >
              <span className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-accent/10 text-accent">
                <Icon className="size-5" aria-hidden />
              </span>
              <div className="flex min-w-0 flex-col-reverse">
                <dt className="text-sm leading-snug text-muted-foreground">{stat.label || t(`labels.${stat.key}`)}</dt>
                <dd className="text-2xl font-bold text-primary tabular-nums" dir="auto">{formatStat(stat, locale)}</dd>
              </div>
            </div>
          );
        })}
      </dl>
    </section>
  );
}
