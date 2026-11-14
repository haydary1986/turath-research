"use client";

import { useEffect, useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import { RotateCcw, Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { DataOption } from "@/components/i18n/data-option";
import { NativeSelect } from "@/components/portal/form-controls";
import {
  PUBLICATION_SORTS, activeFilterCount, type PublicationFilters,
} from "@/lib/publications/filters";
import type { Org } from "@/lib/portal/types";

export interface FacetCount {
  value: string;
  count: number;
}

interface FilterBarProps {
  filters: PublicationFilters;
  years: FacetCount[];
  types: FacetCount[];
  colleges: Org[];
  departments: Org[];
  onChange: (patch: Partial<Omit<PublicationFilters, "page">>) => void;
  onReset: () => void;
}

const TEXT_DEBOUNCE_MS = 400;

/** A text filter that reports its value after the user pauses typing. */
function DebouncedText({ id, label, value, placeholder, onCommit, icon }: {
  id: string; label: string; value: string; placeholder: string; onCommit: (v: string) => void; icon?: boolean;
}) {
  const [draft, setDraft] = useState(value);
  // Follow a value changed from outside (reset, back button) without an effect.
  const [seen, setSeen] = useState(value);
  if (value !== seen) {
    setSeen(value);
    setDraft(value);
  }
  useEffect(() => {
    if (draft === value) return;
    const timer = setTimeout(() => onCommit(draft), TEXT_DEBOUNCE_MS);
    return () => clearTimeout(timer);
  }, [draft, value, onCommit]);

  return (
    <div className="space-y-1.5">
      <Label htmlFor={id} className="text-xs text-muted-foreground">{label}</Label>
      <div className="relative">
        {icon && <Search className="absolute start-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden />}
        <Input id={id} type="search" value={draft} placeholder={placeholder} maxLength={200}
          onChange={(e) => setDraft(e.target.value)} className={icon ? "ps-8" : undefined} />
      </div>
    </div>
  );
}

function Select({ id, label, children, ...props }: { id: string; label: string } & React.ComponentProps<"select">) {
  return (
    <div className="space-y-1.5">
      <Label htmlFor={id} className="text-xs text-muted-foreground">{label}</Label>
      <NativeSelect id={id} className="h-9 w-full" {...props}>{children}</NativeSelect>
    </div>
  );
}

/** Search, author, journal, year range, type, college, department and sort for the publications list. */
export function PublicationFilterBar({ filters, years, types, colleges, departments, onChange, onReset }: FilterBarProps) {
  const t = useTranslations("publication.filters");
  const tp = useTranslations("publication");
  const locale = useLocale();
  const active = activeFilterCount(filters);
  const visibleDepartments = departments.filter((d) => !filters.college_id || d.college_id === filters.college_id);

  return (
    <div className="space-y-3">
      <div className="grid gap-3 md:grid-cols-3">
        <DebouncedText id="pub-q" label={t("search")} placeholder={t("searchPlaceholder")} icon
          value={filters.q} onCommit={(q) => onChange({ q })} />
        <DebouncedText id="pub-author" label={t("author")} placeholder={t("authorPlaceholder")}
          value={filters.author} onCommit={(author) => onChange({ author })} />
        <DebouncedText id="pub-journal" label={t("journal")} placeholder={t("journalPlaceholder")}
          value={filters.journal} onCommit={(journal) => onChange({ journal })} />
      </div>
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-6">
        <Select id="pub-year-from" label={t("yearFrom")} value={filters.year_from}
          onChange={(e) => onChange({ year_from: e.target.value })}>
          <option value="">{t("any")}</option>
          {years.map((y) => <option key={y.value} value={y.value}>{y.value} ({y.count})</option>)}
        </Select>
        <Select id="pub-year-to" label={t("yearTo")} value={filters.year_to}
          onChange={(e) => onChange({ year_to: e.target.value })}>
          <option value="">{t("any")}</option>
          {years.filter((y) => !filters.year_from || y.value >= filters.year_from)
            .map((y) => <option key={y.value} value={y.value}>{y.value}</option>)}
        </Select>
        <Select id="pub-type" label={tp("type")} value={filters.type} onChange={(e) => onChange({ type: e.target.value })}>
          <option value="">{t("allTypes")}</option>
          {types.map((ty) => (
            <option key={ty.value} value={ty.value}>
              {tp.has(`types.${ty.value}`) ? tp(`types.${ty.value}`) : ty.value} ({ty.count})
            </option>
          ))}
        </Select>
        <Select id="pub-college" label={t("college")} value={filters.college_id}
          onChange={(e) => onChange({ college_id: e.target.value })}>
          <option value="">{t("allColleges")}</option>
          {colleges.map((c) => <DataOption key={c.id} value={c.id} ar={c.name_ar} en={c.name_en} locale={locale} />)}
        </Select>
        <Select id="pub-department" label={t("department")} value={filters.department_id}
          onChange={(e) => onChange({ department_id: e.target.value })}>
          <option value="">{t("allDepartments")}</option>
          {visibleDepartments.map((d) => <DataOption key={d.id} value={d.id} ar={d.name_ar} en={d.name_en} locale={locale} />)}
        </Select>
        <Select id="pub-sort" label={t("sort")} value={filters.sort}
          onChange={(e) => onChange({ sort: e.target.value as PublicationFilters["sort"] })}>
          {PUBLICATION_SORTS.map((s) => <option key={s} value={s}>{t(`sorts.${s}`)}</option>)}
        </Select>
      </div>
      {active > 0 && (
        <div className="flex items-center justify-between gap-2 text-sm">
          <span className="text-muted-foreground">{t("activeCount", { count: active })}</span>
          <Button type="button" variant="ghost" size="sm" onClick={onReset}>
            <RotateCcw className="size-4" aria-hidden />
            {t("reset")}
          </Button>
        </div>
      )}
    </div>
  );
}
