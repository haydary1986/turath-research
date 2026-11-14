"use client";

import { useEffect, useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import { Search } from "lucide-react";
import { api } from "@/lib/api";
import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { NativeSelect } from "@/components/portal/form-controls";
import { formatBaghdadTime, loginLogQuery, type LoginEvent, type LoginLogFilters } from "@/lib/portal/audit";
import { Bidi, Ltr } from "@/components/i18n/bidi";

const PAGE_SIZE = 25;
const EMPTY: LoginLogFilters = { q: "", method: "", result: "", from: "", to: "" };

/** The admin's filterable, paginated sign-in log. */
export function LoginLog() {
  const tu = useTranslations("portal.audit");
  const t = useTranslations("portal");
  const locale = useLocale();
  const [filters, setFilters] = useState<LoginLogFilters>(EMPTY);
  const [page, setPage] = useState(1);
  const [items, setItems] = useState<LoginEvent[]>([]);
  const [total, setTotal] = useState(0);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    let active = true;
    const timer = setTimeout(() => {
      api
        .get<LoginEvent[]>(`/api/v1/admin/login-events?${loginLogQuery(filters, page, PAGE_SIZE)}`)
        .then((res) => {
          if (!active) return;
          setItems(res.data ?? []);
          setTotal(res.meta?.total ?? 0);
          setFailed(false);
        })
        .catch(() => active && setFailed(true));
    }, 250);
    return () => {
      active = false;
      clearTimeout(timer);
    };
  }, [filters, page]);

  function change(patch: Partial<LoginLogFilters>) {
    setFilters((prev) => ({ ...prev, ...patch }));
    setPage(1);
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-3">
        <div className="relative min-w-56 flex-1">
          <Search className="absolute start-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input className="ps-10" placeholder={tu("search")} value={filters.q} onChange={(e) => change({ q: e.target.value })} />
        </div>
        <NativeSelect className="w-auto" aria-label={tu("columns.method")} value={filters.method}
          onChange={(e) => change({ method: e.target.value })}>
          <option value="">{tu("allMethods")}</option>
          <option value="password">{tu("methods.password")}</option>
          <option value="google">{tu("methods.google")}</option>
        </NativeSelect>
        <NativeSelect className="w-auto" aria-label={tu("columns.result")} value={filters.result}
          onChange={(e) => change({ result: e.target.value })}>
          <option value="">{tu("allResults")}</option>
          <option value="success">{tu("results.success")}</option>
          <option value="failure">{tu("results.failure")}</option>
        </NativeSelect>
        <Input className="w-auto" type="date" dir="ltr" aria-label={t("fields.from")} value={filters.from}
          onChange={(e) => change({ from: e.target.value })} />
        <Input className="w-auto" type="date" dir="ltr" aria-label={t("fields.to")} value={filters.to}
          onChange={(e) => change({ to: e.target.value })} />
      </div>
      {failed ? (
        <Alert variant="danger">{tu("loadFailed")}</Alert>
      ) : items.length === 0 ? (
        <p className="py-8 text-center text-sm text-muted-foreground">{tu("empty")}</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{tu("columns.time")}</TableHead>
              <TableHead>{tu("columns.user")}</TableHead>
              <TableHead>{tu("columns.method")}</TableHead>
              <TableHead>{tu("columns.result")}</TableHead>
              <TableHead>{tu("columns.ip")}</TableHead>
              <TableHead>{tu("columns.agent")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((e) => (
              <TableRow key={e.id}>
                <TableCell className="whitespace-nowrap"><Ltr>{formatBaghdadTime(e.created_at, locale)}</Ltr></TableCell>
                <TableCell>
                  <Bidi block text={e.user_name} className="font-medium" fallback="—" />
                  <Ltr className="block text-xs text-muted-foreground">{e.email || "—"}</Ltr>
                </TableCell>
                <TableCell>{tu(`methods.${e.method}`)}</TableCell>
                <TableCell>
                  <Badge variant={e.result === "success" ? "secondary" : "destructive"}>{tu(`results.${e.result}`)}</Badge>
                  {e.reason && (
                    <span className="block text-xs text-muted-foreground">
                      {tu.has(`reasons.${e.reason}`) ? tu(`reasons.${e.reason}`) : <Bidi text={e.reason} />}
                    </span>
                  )}
                </TableCell>
                <TableCell><Ltr>{e.ip || "—"}</Ltr></TableCell>
                <TableCell className="max-w-56 truncate text-xs text-muted-foreground" title={e.user_agent} dir="ltr" data-bidi="">
                  {e.user_agent || "—"}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
      {total > PAGE_SIZE && (
        <div className="flex items-center justify-center gap-2">
          <Button variant="outline" size="sm" disabled={page === 1} onClick={() => setPage((p) => p - 1)}>{t("ui.previous")}</Button>
          <span className="text-sm">{page}</span>
          <Button variant="outline" size="sm" disabled={page * PAGE_SIZE >= total} onClick={() => setPage((p) => p + 1)}>
            {t("ui.next")}
          </Button>
        </div>
      )}
    </div>
  );
}
