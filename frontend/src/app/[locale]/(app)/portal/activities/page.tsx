"use client";

import { Suspense, useCallback, useEffect, useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import { CalendarDays, FileSpreadsheet, Plus, Search } from "lucide-react";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useHydrated } from "@/hooks/use-hydrated";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table";
import { ActivityReviewDialog } from "@/components/portal/activity-review-dialog";
import { NativeSelect } from "@/components/portal/form-controls";
import {
  ActivityForm, activityPayload, activityToForm, emptyActivityForm, type ActivityFormState,
} from "@/components/portal/activity-form";
import { FS003Export } from "@/components/portal/fs003-export";
import { RequireAuth } from "@/components/portal/require-auth";
import { usePortalErrors } from "@/components/portal/use-portal-errors";
import { formatDate, isStaffRole, type Org } from "@/lib/portal/types";
import { Alert } from "@/components/ui/alert";
import {
  ACTIVITY_CATEGORIES, activityStatusKey, canExportFS003, currentDecisionNote, enterableCategories,
  isCategory, type Activity,
} from "@/lib/portal/activity";
import { useMyResponsibilities } from "@/lib/portal/my-access";
import { ACTIVITY_STATUS_FAMILY, FAMILY_TINT } from "@/lib/portal/status";
import { canManageSite, type StaffAnnouncement } from "@/lib/site/types";
import { useRouter } from "@/i18n/routing";
import { useSearchParams } from "next/navigation";

const STATUSES = ["pending", "approved", "returned", "rejected"] as const;
const STEPS = ["register", "review", "report"] as const;
const PAGE_SIZE = 20;
const SKELETON_ROWS = [0, 1, 2, 3, 4];

const STATUS_TINT: Record<string, string> = Object.fromEntries(
  Object.entries(ACTIVITY_STATUS_FAMILY).map(([status, family]) => [status, FAMILY_TINT[family]])
);

function ActivitiesPage() {
  const t = useTranslations("portal");
  const ta = useTranslations("portal.activities");
  const toMessage = usePortalErrors();
  const locale = useLocale();
  const hydrated = useHydrated();
  const { user } = useAuth();
  const staff = isStaffRole(user?.role);
  const responsibilities = useMyResponsibilities();
  const categories = enterableCategories(user?.role, responsibilities);
  const canExport = hydrated && canExportFS003(user?.role, responsibilities);
  const searchParams = useSearchParams();
  const initialCategory = searchParams.get("category") ?? "";

  const [items, setItems] = useState<Activity[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadFailed, setLoadFailed] = useState(false);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [filters, setFilters] = useState({
    q: "", category: isCategory(initialCategory) ? initialCategory : "", status: "", year: "", mine: false,
  });
  const [exporting, setExporting] = useState(false);
  const [colleges, setColleges] = useState<Org[]>([]);
  const [departments, setDepartments] = useState<Org[]>([]);
  const [form, setForm] = useState<ActivityFormState | null>(null);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const load = useCallback(() => {
    const params = new URLSearchParams({ page: String(page), limit: String(PAGE_SIZE) });
    if (filters.q.trim()) params.set("q", filters.q.trim());
    if (filters.category) params.set("category", filters.category);
    if (filters.mine) params.set("mine", "true");
    if (filters.status) params.set("status", filters.status);
    if (filters.year) params.set("year", filters.year);
    api
      .get<Activity[]>(`/api/v1/activities?${params}`)
      .then((res) => {
        setItems(res.data ?? []);
        setTotal(res.meta?.total ?? 0);
        setLoadFailed(false);
      })
      .catch(() => {
        setItems([]);
        setLoadFailed(true);
      })
      .finally(() => setLoading(false));
  }, [page, filters]);

  useEffect(() => {
    const timer = setTimeout(load, 250);
    return () => clearTimeout(timer);
  }, [load]);

  useEffect(() => {
    Promise.all([api.get<Org[]>("/api/v1/colleges"), api.get<Org[]>("/api/v1/departments")])
      .then(([c, d]) => {
        setColleges(c.data ?? []);
        setDepartments(d.data ?? []);
      })
      .catch(() => undefined);
  }, []);

  function startEdit(a: Activity) {
    setEditingId(a.id);
    setForm(activityToForm(a));
  }

  function startNew() {
    setEditingId(null);
    if (form) {
      setForm(null);
      return;
    }
    const preferred = filters.category && categories.includes(filters.category as never)
      ? filters.category
      : categories.includes("workshops") ? "workshops" : categories[0];
    setForm(emptyActivityForm(preferred, user?.college_id ?? "", user?.department_id ?? ""));
  }

  async function save() {
    if (!form) return;
    setError("");
    setBusy(true);
    try {
      if (editingId) {
        await api.put(`/api/v1/activities/${editingId}`, activityPayload(form));
      } else {
        await api.post("/api/v1/activities", activityPayload(form));
      }
      setForm(null);
      setEditingId(null);
      load();
    } catch (err) {
      setError(toMessage(err));
    } finally {
      setBusy(false);
    }
  }

  const editingReturned = items.some((a) => a.id === editingId && a.status === "returned");
  const yearOptions = [0, 1, 2].map((d) => String(new Date().getFullYear() - d));

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold">{ta("title")}</h1>
          <p className="text-sm text-muted-foreground">{ta("intro")}</p>
        </div>
        <div className="flex flex-wrap gap-2">
          {canExport && (
            <Button variant="outline" onClick={() => setExporting((v) => !v)}>
              <FileSpreadsheet className="size-4" />
              <span>{ta("fs003.open")}</span>
            </Button>
          )}
          <Button onClick={startNew}>
            <Plus className="size-4" />
            <span>{ta("new")}</span>
          </Button>
        </div>
      </div>

      <ol className="grid gap-3 sm:grid-cols-3">
        {STEPS.map((step, index) => (
          <li key={step} className="flex items-center gap-3 rounded-lg border bg-card p-3 text-sm">
            <span className="flex size-7 shrink-0 items-center justify-center rounded-full bg-primary/10 font-semibold text-primary">
              {index + 1}
            </span>
            <span>{ta(`steps.${step}`)}</span>
          </li>
        ))}
      </ol>

      {error && <Alert variant="danger">{error}</Alert>}

      {exporting && <FS003Export colleges={colleges} onClose={() => setExporting(false)} />}

      {form && (
        <ActivityForm
          form={form}
          editing={editingId !== null}
          editingReturned={editingReturned}
          categories={categories}
          colleges={colleges}
          departments={departments}
          busy={busy}
          onChange={setForm}
          onSave={save}
          onCancel={() => { setForm(null); setEditingId(null); }}
        />
      )}

      <Card>
        <CardHeader>
          <div className="flex flex-wrap items-center gap-3">
            <div className="relative min-w-56 flex-1">
              <Search className="absolute start-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input className="ps-10" placeholder={ta("searchPlaceholder")} value={filters.q}
                onChange={(e) => { setFilters({ ...filters, q: e.target.value }); setPage(1); }} />
            </div>
            <NativeSelect className="w-auto max-w-64" value={filters.category}
              onChange={(e) => { setFilters({ ...filters, category: e.target.value }); setPage(1); }}>
              <option value="">{ta("allCategories")}</option>
              {ACTIVITY_CATEGORIES.map((x) => <option key={x} value={x}>{ta(`categories.${x}`)}</option>)}
            </NativeSelect>
            <NativeSelect className="w-auto" value={filters.status}
              onChange={(e) => { setFilters({ ...filters, status: e.target.value }); setPage(1); }}>
              <option value="">{t("ui.allStatuses")}</option>
              {STATUSES.map((s) => <option key={s} value={s}>{ta(`statuses.${s}`)}</option>)}
            </NativeSelect>
            <NativeSelect className="w-auto" value={filters.year}
              onChange={(e) => { setFilters({ ...filters, year: e.target.value }); setPage(1); }}>
              <option value="">{ta("allYears")}</option>
              {yearOptions.map((y) => <option key={y} value={y}>{y}</option>)}
            </NativeSelect>
            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={filters.mine}
                onChange={(e) => { setFilters({ ...filters, mine: e.target.checked }); setPage(1); }} />
              {ta("mine")}
            </label>
          </div>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div role="status" aria-busy="true" className="space-y-3 py-2">
              <span className="sr-only">{t("ui.loading")}</span>
              {SKELETON_ROWS.map((row) => <Skeleton key={row} className="h-10 w-full" />)}
            </div>
          ) : loadFailed ? (
            <Alert variant="danger">{ta("loadFailed")}</Alert>
          ) : items.length === 0 ? (
            <p className="py-8 text-center text-sm text-muted-foreground">{ta("empty")}</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{ta("fields2.category")}</TableHead>
                  <TableHead>{ta("fields.title")}</TableHead>
                  <TableHead>{t("fields.college")}</TableHead>
                  <TableHead>{ta("fields.date")}</TableHead>
                  <TableHead>{ta("fields.attendees")}</TableHead>
                  <TableHead>{ta("fields.approvalNo")}</TableHead>
                  <TableHead>{t("fields.status")}</TableHead>
                  <TableHead>{ta("actions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((a) => (
                  <ActivityRow
                    key={a.id}
                    activity={a}
                    viewerIsOwner={hydrated && user?.id === a.created_by}
                    canEdit={hydrated && a.can_edit}
                    canDecide={
                      hydrated && staff && (a.status === "pending" || a.status === "returned") &&
                      (user?.role === "admin" || user?.id !== a.created_by)
                    }
                    canAnnounce={hydrated && canManageSite(user?.role) && a.status === "approved"}
                    locale={locale}
                    onEdit={() => startEdit(a)}
                    onChanged={load}
                  />
                ))}
              </TableBody>
            </Table>
          )}
          {total > PAGE_SIZE && (
            <div className="mt-4 flex items-center justify-center gap-2">
              <Button variant="outline" size="sm" disabled={page === 1} onClick={() => setPage((p) => p - 1)}>
                {t("ui.previous")}
              </Button>
              <span className="text-sm">{page}</span>
              <Button variant="outline" size="sm" disabled={page * PAGE_SIZE >= total} onClick={() => setPage((p) => p + 1)}>
                {t("ui.next")}
              </Button>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

interface ActivityRowProps {
  activity: Activity;
  viewerIsOwner: boolean;
  canEdit: boolean;
  canDecide: boolean;
  canAnnounce: boolean;
  locale: string;
  onEdit: () => void;
  onChanged: () => void;
}

function ActivityRow({ activity: a, viewerIsOwner, canEdit, canDecide, canAnnounce, locale, onEdit, onChanged }: ActivityRowProps) {
  const t = useTranslations("portal");
  const ta = useTranslations("portal.activities");
  const toMessage = usePortalErrors();
  const router = useRouter();
  const [reviewing, setReviewing] = useState(false);
  const [announcing, setAnnouncing] = useState(false);

  // Drafts an unpublished announcement from the approved activity and opens it for review.
  async function announce() {
    setAnnouncing(true);
    try {
      const res = await api.post<StaffAnnouncement>(`/api/v1/activities/${a.id}/announce`, {});
      if (res.data) router.push(`/portal/site?edit=${res.data.id}`);
    } catch (err) {
      window.alert(toMessage(err));
    } finally {
      setAnnouncing(false);
    }
  }
  const note = currentDecisionNote(a.status, a.decision_note);

  return (
    <TableRow>
      <TableCell className="max-w-48">
        <span className="block font-medium">{ta(`categories.${a.category}`)}</span>
        {a.category === "workshops" && (
          <span className="block text-xs text-muted-foreground">{ta(`types.${a.activity_type}`)}</span>
        )}
      </TableCell>
      <TableCell className="max-w-xs">
        <span className="block truncate font-medium" title={a.title_ar}>{a.title_ar}</span>
        <span className="block truncate text-xs text-muted-foreground">
          {a.creator_name}{a.presenter ? ` — ${a.presenter}` : ""}
          {a.category === "infrastructure" ? ` — ${a.item_count ?? ""}` : a.category_detail ? ` — ${a.category_detail}` : ""}
        </span>
      </TableCell>
      <TableCell>{a.college_name ?? "—"}</TableCell>
      <TableCell className="whitespace-nowrap">
        <span dir="ltr" className="inline-flex items-center gap-1">
          <CalendarDays className="size-3.5 text-muted-foreground" />
          {formatDate(a.activity_date, locale)}
        </span>
      </TableCell>
      <TableCell className="tabular-nums">{a.attendees_count}</TableCell>
      <TableCell><span dir="ltr">{a.approval_no ?? "—"}</span></TableCell>
      <TableCell className="max-w-64 whitespace-normal">
        <Badge variant="secondary" className={`h-auto whitespace-normal ${STATUS_TINT[a.status] ?? ""}`}>
          {ta(`statuses.${activityStatusKey(a.status, viewerIsOwner)}`)}
        </Badge>
        {note && (
          <p className="mt-1 text-xs whitespace-pre-line break-words text-muted-foreground">
            <span className="font-medium">{ta("decisionNote")}:</span> {note}
          </p>
        )}
      </TableCell>
      <TableCell>
        <div className="flex items-center gap-2">
          {canEdit && <Button size="xs" variant="outline" onClick={onEdit}>{t("ui.edit")}</Button>}
          {canDecide && <Button size="xs" onClick={() => setReviewing(true)}>{ta("reviewAction")}</Button>}
          {canAnnounce && (
            <Button size="xs" variant="outline" disabled={announcing} onClick={announce}>{ta("announce")}</Button>
          )}
          {!canEdit && !canDecide && !canAnnounce && <span className="text-muted-foreground">—</span>}
        </div>
        {reviewing && (
          <ActivityReviewDialog
            activity={a}
            locale={locale}
            onClose={() => setReviewing(false)}
            onDecided={() => { setReviewing(false); onChanged(); }}
          />
        )}
      </TableCell>
    </TableRow>
  );
}

export default function Page() {
  return (
    <RequireAuth>
      <Suspense>
        <ActivitiesPage />
      </Suspense>
    </RequireAuth>
  );
}
