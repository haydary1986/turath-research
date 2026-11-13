"use client";

import { useEffect, useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import { ArrowDown, ArrowUp, RotateCcw } from "lucide-react";
import { api } from "@/lib/api";
import { formatDateTime } from "@/lib/portal/types";
import { formatNumber } from "@/lib/site/format";
import { SITE_ADMIN_API, SITE_LIMITS, type StaffStat, type StatKey, type StatSetting } from "@/lib/site/types";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { Field, move, replaceAt } from "./editor-parts";

interface StaffStatsResponse {
  items: StaffStat[];
  computed_at: string;
}

interface StatsEditorProps {
  stats: StatSetting[];
  onChange: (stats: StatSetting[]) => void;
}

/** Statistics bar: per statistic a label, an optional manual value, visibility and order. */
export function StatsEditor({ stats, onChange }: StatsEditorProps) {
  const t = useTranslations("siteEditor.stats");
  const tl = useTranslations("landing.stats.labels");
  const tc = useTranslations("portal.site.content");
  const locale = useLocale();
  const [computed, setComputed] = useState<StaffStatsResponse | null>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    let active = true;
    api
      .get<StaffStatsResponse>(SITE_ADMIN_API.stats)
      .then((res) => active && res.data && setComputed(res.data))
      .catch(() => active && setFailed(true));
    return () => {
      active = false;
    };
  }, []);

  const numberOf = (key: StatKey): number | null => computed?.items.find((s) => s.key === key)?.computed ?? null;
  const set = (i: number, patch: Partial<StatSetting>) => onChange(replaceAt(stats, i, { ...stats[i], ...patch }));
  const visible = stats.filter((s) => !s.hidden);

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("title")}</CardTitle>
        <p className="text-sm text-muted-foreground">{t("intro")}</p>
        {computed && <p className="text-xs text-muted-foreground">{t("computedAt", { date: formatDateTime(computed.computed_at, locale) })}</p>}
      </CardHeader>
      <CardContent className="space-y-3">
        {failed && <Alert variant="warning">{t("loadFailed")}</Alert>}
        {stats.map((stat, i) => {
          const auto = numberOf(stat.key);
          const manual = stat.value !== "";
          return (
            <fieldset key={stat.key} className={cn("space-y-3 rounded-lg border p-3", stat.hidden && "bg-muted/50")}>
              <legend className="sr-only">{tl(stat.key)}</legend>
              <div className="flex flex-wrap items-center gap-2">
                <p className="font-semibold">{tl(stat.key)}</p>
                <span className={cn("rounded-full px-2 py-0.5 text-xs", manual ? "bg-accent/10 text-accent" : "bg-muted text-muted-foreground")}>
                  {manual ? t("manualBadge") : t("autoBadge")}
                </span>
                <div className="ms-auto flex gap-1">
                  <Button type="button" variant="ghost" size="icon-sm" aria-label={tc("moveUp")} disabled={i === 0}
                    onClick={() => onChange(move(stats, i, i - 1))}>
                    <ArrowUp aria-hidden />
                  </Button>
                  <Button type="button" variant="ghost" size="icon-sm" aria-label={tc("moveDown")} disabled={i === stats.length - 1}
                    onClick={() => onChange(move(stats, i, i + 1))}>
                    <ArrowDown aria-hidden />
                  </Button>
                </div>
              </div>
              <p className="text-xs text-muted-foreground">{t(`what.${stat.key}`)}</p>
              <div className="grid gap-3 sm:grid-cols-2">
                <Field id={`stat_label_${stat.key}`} label={t("label")}>
                  <Input id={`stat_label_${stat.key}`} maxLength={SITE_LIMITS.statLabel} value={stat.label}
                    placeholder={t("labelPlaceholder", { label: tl(stat.key) })} onChange={(e) => set(i, { label: e.target.value })} />
                </Field>
                <Field id={`stat_value_${stat.key}`} label={t("value")}
                  hint={auto !== null ? t("automatic", { value: formatNumber(auto, locale) }) : undefined}>
                  <div className="flex gap-2">
                    <Input id={`stat_value_${stat.key}`} dir="auto" maxLength={SITE_LIMITS.statValue} value={stat.value}
                      placeholder={t("valuePlaceholder")} onChange={(e) => set(i, { value: e.target.value })} />
                    <Button type="button" variant="outline" size="sm" className="h-8 shrink-0" disabled={!manual}
                      onClick={() => set(i, { value: "" })}>
                      <RotateCcw aria-hidden />
                      {t("useAutomatic")}
                    </Button>
                  </div>
                </Field>
              </div>
              <label className="flex items-center gap-2 text-sm">
                <input type="checkbox" className="size-4 accent-primary" checked={!stat.hidden}
                  onChange={(e) => set(i, { hidden: !e.target.checked })} />
                {t("visible")}
              </label>
            </fieldset>
          );
        })}

        {visible.length > 0 && (
          <div>
            <p className="mb-2 text-sm font-semibold">{t("preview")}</p>
            <dl className="flex flex-wrap gap-px overflow-hidden rounded-xl border bg-border">
              {visible.map((stat) => {
                const auto = numberOf(stat.key);
                return (
                  <div key={stat.key} className="flex min-w-[calc(50%-1px)] flex-1 flex-col-reverse bg-card px-3 py-3 sm:min-w-0">
                    <dt className="text-xs text-muted-foreground">{stat.label || tl(stat.key)}</dt>
                    <dd className="text-lg font-bold text-primary tabular-nums" dir="auto">
                      {stat.value || (auto !== null ? formatNumber(auto, locale) : "—")}
                    </dd>
                  </div>
                );
              })}
            </dl>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
