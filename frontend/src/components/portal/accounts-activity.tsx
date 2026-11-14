"use client";

import { useEffect, useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import { api } from "@/lib/api";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { formatBaghdadTime, type AccountActivity, type AccountSummary } from "@/lib/portal/audit";
import { cn } from "@/lib/utils";
import { Bidi, Ltr } from "@/components/i18n/bidi";

const PAGE_SIZE = 25;

/** Online now, active today / in 30 days, never signed in, and each account's last activity. */
export function AccountsActivity() {
  const tu = useTranslations("portal.audit");
  const t = useTranslations("portal");
  const locale = useLocale();
  const [summary, setSummary] = useState<AccountSummary | null>(null);
  const [status, setStatus] = useState("");
  const [q, setQ] = useState("");
  const [page, setPage] = useState(1);
  const [items, setItems] = useState<AccountActivity[]>([]);
  const [total, setTotal] = useState(0);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    api.get<AccountSummary>("/api/v1/admin/accounts/summary").then((res) => setSummary(res.data ?? null)).catch(() => setFailed(true));
  }, []);

  useEffect(() => {
    let active = true;
    const timer = setTimeout(() => {
      const params = new URLSearchParams({ page: String(page), limit: String(PAGE_SIZE) });
      if (status) params.set("status", status);
      if (q.trim()) params.set("q", q.trim());
      api
        .get<AccountActivity[]>(`/api/v1/admin/accounts?${params}`)
        .then((res) => {
          if (!active) return;
          setItems(res.data ?? []);
          setTotal(res.meta?.total ?? 0);
        })
        .catch(() => active && setFailed(true));
    }, 250);
    return () => {
      active = false;
      clearTimeout(timer);
    };
  }, [status, q, page]);

  const tiles: { key: string; label: string; value?: number }[] = [
    { key: "", label: tu("summary.total"), value: summary?.total },
    { key: "online", label: tu("summary.online"), value: summary?.online },
    { key: "today", label: tu("summary.today"), value: summary?.today },
    { key: "30d", label: tu("summary.last30"), value: summary?.last_30_days },
    { key: "never", label: tu("summary.never"), value: summary?.never_signed_in },
  ];

  return (
    <div className="space-y-4">
      <div className="grid gap-3 sm:grid-cols-5" role="group" aria-label={tu("tabs.accounts")}>
        {tiles.map((tile) => (
          <button key={tile.key} type="button" aria-pressed={status === tile.key}
            onClick={() => { setStatus(tile.key); setPage(1); }}
            className={cn("rounded-lg text-start outline-none focus-visible:ring-2 focus-visible:ring-ring",
              status === tile.key && "ring-2 ring-primary")}>
            <Card>
              <CardContent className="py-3">
                <p className="text-xs text-muted-foreground">{tile.label}</p>
                <p className="text-2xl font-bold tabular-nums">{tile.value ?? "—"}</p>
              </CardContent>
            </Card>
          </button>
        ))}
      </div>
      <p className="text-xs text-muted-foreground">{tu("onlineHint")}</p>
      <Input placeholder={tu("search")} value={q} onChange={(e) => { setQ(e.target.value); setPage(1); }} />
      {failed && <Alert variant="danger">{tu("loadFailed")}</Alert>}
      {items.length === 0 ? (
        <p className="py-8 text-center text-sm text-muted-foreground">{tu("empty")}</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{tu("columns.user")}</TableHead>
              <TableHead>{tu("columns.role")}</TableHead>
              <TableHead>{tu("columns.college")}</TableHead>
              <TableHead>{tu("columns.lastSeen")}</TableHead>
              <TableHead>{tu("columns.lastLogin")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((a) => (
              <TableRow key={a.id}>
                <TableCell>
                  <div className="flex items-center gap-2 font-medium">
                    <span aria-hidden className={cn("size-2 rounded-full", a.online ? "bg-status-success-fg" : "bg-muted-foreground/30")} />
                    <Bidi text={a.name_ar} fallback="—" />
                    {a.online && <span className="sr-only">{tu("summary.online")}</span>}
                  </div>
                  <Ltr className="block text-xs text-muted-foreground">{a.email}</Ltr>
                </TableCell>
                <TableCell>{t.has(`users.roleNames.${a.role}`) ? t(`users.roleNames.${a.role}`) : a.role}</TableCell>
                <TableCell><Bidi block text={a.college_name} fallback="—" /></TableCell>
                <TableCell className="whitespace-nowrap">
                  {a.last_seen_at ? <Ltr>{formatBaghdadTime(a.last_seen_at, locale)}</Ltr> : tu("never")}
                </TableCell>
                <TableCell className="whitespace-nowrap">
                  {a.last_login_at ? <Ltr>{formatBaghdadTime(a.last_login_at, locale)}</Ltr> : tu("never")}
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
