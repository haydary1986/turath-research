"use client";

import { Suspense, useCallback, useEffect, useMemo, useState } from "react";
import { useSearchParams } from "next/navigation";
import { useTranslations } from "next-intl";
import { usePathname, useRouter } from "@/i18n/routing";
import { api } from "@/lib/api";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { PublicationFilterBar, type FacetCount } from "@/components/publications/publication-filter-bar";
import { PublicationsTable, type PublicationItem } from "@/components/publications/publications-table";
import {
  EMPTY_FILTERS, PAGE_SIZE, filtersFromParams, filtersToApiQuery, filtersToParams, withFilter,
  type PublicationFilters,
} from "@/lib/publications/filters";
import type { Org } from "@/lib/portal/types";

interface ListResult {
  /** The API query this result answers. */
  query: string;
  items: PublicationItem[];
  total: number;
  failed: boolean;
}

interface Facets {
  years: FacetCount[];
  types: FacetCount[];
}

/** Years, types, colleges and departments for the filter menus, loaded once. */
function useFilterOptions() {
  const [facets, setFacets] = useState<Facets>({ years: [], types: [] });
  const [colleges, setColleges] = useState<Org[]>([]);
  const [departments, setDepartments] = useState<Org[]>([]);

  useEffect(() => {
    let active = true;
    Promise.all([
      api.get<Facets>("/api/v1/publications/facets"),
      api.get<Org[]>("/api/v1/colleges"),
      api.get<Org[]>("/api/v1/departments"),
    ])
      .then(([f, c, d]) => {
        if (!active) return;
        setFacets(f.data ?? { years: [], types: [] });
        setColleges(c.data ?? []);
        setDepartments(d.data ?? []);
      })
      .catch(() => undefined); // the list still works without the menus
    return () => {
      active = false;
    };
  }, []);

  return { facets, colleges, departments };
}

function PublicationsBrowser() {
  const t = useTranslations("publication");
  const ct = useTranslations("common");
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const filters = useMemo(() => filtersFromParams(new URLSearchParams(searchParams.toString())), [searchParams]);
  const { facets, colleges, departments } = useFilterOptions();

  const apiQuery = filtersToApiQuery(filters);
  const [result, setResult] = useState<ListResult>({ query: "", items: [], total: 0, failed: false });
  useEffect(() => {
    let active = true;
    api
      .get<PublicationItem[]>(`/api/v1/publications?${apiQuery}`)
      .then((res) => active && setResult({ query: apiQuery, items: res.data ?? [], total: res.meta?.total ?? 0, failed: false }))
      .catch(() => active && setResult((prev) => ({ ...prev, query: apiQuery, failed: true })));
    return () => {
      active = false;
    };
  }, [apiQuery]);
  // Until the answer for the current query arrives, the previous rows stay visible, dimmed.
  const loading = result.query !== apiQuery;
  const { items, total, failed } = result;

  const navigate = useCallback(
    (next: PublicationFilters) => {
      const query = filtersToParams(next).toString();
      router.replace(query ? `${pathname}?${query}` : pathname, { scroll: false });
    },
    [router, pathname]
  );
  const onChange = useCallback(
    (patch: Partial<Omit<PublicationFilters, "page">>) => navigate(withFilter(filters, patch)),
    [filters, navigate]
  );

  const pages = Math.max(1, Math.ceil(total / PAGE_SIZE));
  // Latin digits in both languages, as everywhere else in the portal.
  const formatted = new Intl.NumberFormat("en-US").format(total);

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-2">
        <div>
          <h1 className="text-2xl font-bold">{t("title")}</h1>
          <p className="text-sm text-muted-foreground">{t("filters.subtitle")}</p>
        </div>
        <p className="text-sm text-muted-foreground" aria-live="polite">{t("filters.resultCount", { count: total, formatted })}</p>
      </div>

      <Card>
        <CardHeader>
          <PublicationFilterBar filters={filters} years={facets.years} types={facets.types} colleges={colleges}
            departments={departments} onChange={onChange} onReset={() => navigate(EMPTY_FILTERS)} />
        </CardHeader>
        <CardContent>
          {failed ? (
            <Alert variant="danger">{t("filters.loadFailed")}</Alert>
          ) : loading && items.length === 0 ? (
            <p className="py-8 text-center text-muted-foreground">{ct("loading")}</p>
          ) : items.length === 0 ? (
            <p className="py-8 text-center text-muted-foreground">{ct("noResults")}</p>
          ) : (
            <div className={loading ? "opacity-60 transition-opacity" : undefined}>
              <PublicationsTable items={items} />
            </div>
          )}

          {pages > 1 && !failed && (
            <nav aria-label={t("filters.pagination")} className="mt-4 flex items-center justify-center gap-3">
              <Button variant="outline" size="sm" disabled={filters.page <= 1}
                onClick={() => navigate({ ...filters, page: filters.page - 1 })}>
                {ct("previous")}
              </Button>
              <span className="text-sm tabular-nums">{t("filters.pageOf", { page: filters.page, pages })}</span>
              <Button variant="outline" size="sm" disabled={filters.page >= pages}
                onClick={() => navigate({ ...filters, page: filters.page + 1 })}>
                {ct("next")}
              </Button>
            </nav>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

export default function PublicationsPage() {
  return (
    <Suspense>
      <PublicationsBrowser />
    </Suspense>
  );
}
