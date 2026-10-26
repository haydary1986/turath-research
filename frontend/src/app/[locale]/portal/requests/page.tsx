"use client";

import { useEffect, useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import { Search } from "lucide-react";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { NativeSelect } from "@/components/portal/form-controls";
import { RequireAuth } from "@/components/portal/require-auth";
import { RequestsTable } from "@/components/portal/requests-table";
import { REQUEST_KINDS, isStaffRole, type PortalRequest, type RequestStatus } from "@/lib/portal/types";

const PAGE_SIZE = 20;
const STATUSES: RequestStatus[] = [
  "draft", "pending_dean", "pending", "pending_vp", "returned", "approved", "rejected",
  "verified", "not_eligible", "awaiting_signatures", "completed", "issued",
];

function RequestsList() {
  const t = useTranslations("portal");
  const locale = useLocale();
  const { user } = useAuth();
  const sees = isStaffRole(user?.role) || user?.role === "department_head" || user?.role === "college_dean";

  const [items, setItems] = useState<PortalRequest[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [q, setQ] = useState("");
  const [kind, setKind] = useState("");
  const [status, setStatus] = useState("");
  const [mine, setMine] = useState(!sees);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let active = true;
    const params = new URLSearchParams({ page: String(page), limit: String(PAGE_SIZE) });
    if (q.trim()) params.set("q", q.trim());
    if (kind) params.set("kind", kind);
    if (status) params.set("status", status);
    if (mine) params.set("mine", "true");

    // Debounce typing in the search box.
    const timer = setTimeout(() => {
      setLoading(true);
      api
        .get<PortalRequest[]>(`/api/v1/portal/requests?${params}`)
        .then((res) => {
          if (!active) return;
          setItems(res.data ?? []);
          setTotal(res.meta?.total ?? 0);
        })
        .catch(() => active && setItems([]))
        .finally(() => active && setLoading(false));
    }, 250);

    return () => {
      active = false;
      clearTimeout(timer);
    };
  }, [page, q, kind, status, mine]);

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{sees && !mine ? t("ui.allRequests") : t("ui.myRequests")}</h1>
      <Card>
        <CardHeader>
          <div className="flex flex-wrap items-center gap-3">
            <div className="relative min-w-60 flex-1">
              <Search className="absolute start-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                className="ps-10"
                placeholder={t("ui.searchPlaceholder")}
                value={q}
                onChange={(e) => {
                  setQ(e.target.value);
                  setPage(1);
                }}
              />
            </div>
            <NativeSelect
              className="w-auto"
              value={kind}
              onChange={(e) => {
                setKind(e.target.value);
                setPage(1);
              }}
            >
              <option value="">{t("ui.allKinds")}</option>
              {REQUEST_KINDS.map((k) => (
                <option key={k} value={k}>
                  {t(`kinds.${k}`)}
                </option>
              ))}
            </NativeSelect>
            <NativeSelect
              className="w-auto"
              value={status}
              onChange={(e) => {
                setStatus(e.target.value);
                setPage(1);
              }}
            >
              <option value="">{t("ui.allStatuses")}</option>
              {STATUSES.map((s) => (
                <option key={s} value={s}>
                  {t(`statuses.${s}`)}
                </option>
              ))}
            </NativeSelect>
            {sees && (
              <label className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={mine}
                  onChange={(e) => {
                    setMine(e.target.checked);
                    setPage(1);
                  }}
                />
                {t("ui.mineOnly")}
              </label>
            )}
          </div>
        </CardHeader>
        <CardContent>
          {loading ? (
            <p className="py-6 text-center text-muted-foreground">{t("ui.loading")}</p>
          ) : (
            <RequestsTable items={items} locale={locale} showApplicant={sees} />
          )}
          {total > PAGE_SIZE && (
            <div className="mt-4 flex items-center justify-center gap-2">
              <Button variant="outline" size="sm" disabled={page === 1} onClick={() => setPage((p) => p - 1)}>
                {t("ui.previous")}
              </Button>
              <span className="text-sm">{page}</span>
              <Button
                variant="outline"
                size="sm"
                disabled={page * PAGE_SIZE >= total}
                onClick={() => setPage((p) => p + 1)}
              >
                {t("ui.next")}
              </Button>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

export default function RequestsPage() {
  return (
    <RequireAuth>
      <RequestsList />
    </RequireAuth>
  );
}
