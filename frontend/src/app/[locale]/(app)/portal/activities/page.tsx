"use client";

import { useCallback, useEffect, useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import { CalendarDays, Plus, Search } from "lucide-react";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useHydrated } from "@/hooks/use-hydrated";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table";
import { ActivityReviewDialog } from "@/components/portal/activity-review-dialog";
import { NativeSelect, Textarea } from "@/components/portal/form-controls";
import { RequireAuth } from "@/components/portal/require-auth";
import { usePortalErrors } from "@/components/portal/use-portal-errors";
import { formatDate, isStaffRole, type Org } from "@/lib/portal/types";
import { Alert } from "@/components/ui/alert";
import { activityStatusKey, currentDecisionNote, type Activity } from "@/lib/portal/activity";
import { ACTIVITY_STATUS_FAMILY, FAMILY_TINT } from "@/lib/portal/status";
import { canManageSite, type StaffAnnouncement } from "@/lib/site/types";
import { useRouter } from "@/i18n/routing";

const TYPES = ["seminar", "workshop", "panel", "lecture", "course", "conference", "other"] as const;
const STATUSES = ["pending", "approved", "returned", "rejected"] as const;
const MODES = ["in_person", "online", "hybrid"] as const;
const STEPS = ["register", "review", "report"] as const;
const PAGE_SIZE = 20;
const SKELETON_ROWS = [0, 1, 2, 3, 4];

const STATUS_TINT: Record<string, string> = Object.fromEntries(
  Object.entries(ACTIVITY_STATUS_FAMILY).map(([status, family]) => [status, FAMILY_TINT[family]])
);

interface FormState {
  activity_type: string;
  title_ar: string;
  description: string;
  presenter: string;
  college_id: string;
  department_id: string;
  activity_date: string;
  location: string;
  attendees_count: number;
  target_audience: string;
  organizer: string;
  attendance_mode: string;
}

const EMPTY_FORM: FormState = {
  activity_type: "seminar",
  title_ar: "",
  description: "",
  presenter: "",
  college_id: "",
  department_id: "",
  activity_date: "",
  location: "",
  attendees_count: 0,
  target_audience: "",
  organizer: "",
  attendance_mode: "",
};

function ActivitiesPage() {
  const t = useTranslations("portal");
  const ta = useTranslations("portal.activities");
  const tc = useTranslations("common");
  const toMessage = usePortalErrors();
  const locale = useLocale();
  const hydrated = useHydrated();
  const { user } = useAuth();
  const staff = isStaffRole(user?.role);

  const [items, setItems] = useState<Activity[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadFailed, setLoadFailed] = useState(false);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [filters, setFilters] = useState({ q: "", type: "", status: "", year: "" });
  const [colleges, setColleges] = useState<Org[]>([]);
  const [departments, setDepartments] = useState<Org[]>([]);
  const [form, setForm] = useState<FormState | null>(null);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const load = useCallback(() => {
    const params = new URLSearchParams({ page: String(page), limit: String(PAGE_SIZE) });
    if (filters.q.trim()) params.set("q", filters.q.trim());
    if (filters.type) params.set("type", filters.type);
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
    setForm({
      activity_type: a.activity_type,
      title_ar: a.title_ar,
      description: a.description ?? "",
      presenter: a.presenter ?? "",
      college_id: a.college_id ?? "",
      department_id: a.department_id ?? "",
      activity_date: a.activity_date,
      location: a.location ?? "",
      attendees_count: a.attendees_count,
      target_audience: a.target_audience ?? "",
      organizer: a.organizer ?? "",
      attendance_mode: a.attendance_mode ?? "",
    });
  }

  async function save() {
    if (!form) return;
    setError("");
    setBusy(true);
    try {
      if (editingId) {
        await api.put(`/api/v1/activities/${editingId}`, form);
      } else {
        await api.post("/api/v1/activities", form);
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
        <Button
          onClick={() => {
            setEditingId(null);
            setForm(form ? null : { ...EMPTY_FORM, college_id: user?.college_id ?? "", department_id: user?.department_id ?? "" });
          }}
        >
          <Plus className="size-4" />
          <span>{ta("new")}</span>
        </Button>
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

      {form && (
        <Card>
          <CardHeader>
            <CardTitle>{editingId ? t("ui.edit") : ta("new")}</CardTitle>
          </CardHeader>
          <CardContent className="grid gap-4 md:grid-cols-2">
            {editingReturned && (
              <Alert variant="info" className="md:col-span-2">{ta("resubmitHint")}</Alert>
            )}
            <div className="space-y-2 md:col-span-2">
              <Label htmlFor="act-title">{ta("fields.title")} *</Label>
              <Input id="act-title" dir="rtl" value={form.title_ar}
                onChange={(e) => setForm({ ...form, title_ar: e.target.value })} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="act-type">{ta("fields.type")} *</Label>
              <NativeSelect id="act-type" value={form.activity_type}
                onChange={(e) => setForm({ ...form, activity_type: e.target.value })}>
                {TYPES.map((x) => <option key={x} value={x}>{ta(`types.${x}`)}</option>)}
              </NativeSelect>
            </div>
            <div className="space-y-2">
              <Label htmlFor="act-date">{ta("fields.date")} *</Label>
              <Input id="act-date" type="date" dir="ltr" value={form.activity_date}
                onChange={(e) => setForm({ ...form, activity_date: e.target.value })} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="act-college">{t("fields.college")} *</Label>
              <NativeSelect id="act-college" value={form.college_id}
                onChange={(e) => setForm({ ...form, college_id: e.target.value, department_id: "" })}>
                <option value="">{t("options.none")}</option>
                {colleges.map((c) => <option key={c.id} value={c.id}>{c.name_ar}</option>)}
              </NativeSelect>
            </div>
            <div className="space-y-2">
              <Label htmlFor="act-dept">{t("fields.department")}</Label>
              <NativeSelect id="act-dept" value={form.department_id}
                onChange={(e) => setForm({ ...form, department_id: e.target.value })}>
                <option value="">{t("options.none")}</option>
                {departments.filter((d) => !form.college_id || d.college_id === form.college_id)
                  .map((d) => <option key={d.id} value={d.id}>{d.name_ar}</option>)}
              </NativeSelect>
            </div>
            <div className="space-y-2">
              <Label htmlFor="act-presenter">{ta("fields.presenter")}</Label>
              <Input id="act-presenter" value={form.presenter}
                onChange={(e) => setForm({ ...form, presenter: e.target.value })} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="act-location">{ta("fields.location")}</Label>
              <Input id="act-location" value={form.location}
                onChange={(e) => setForm({ ...form, location: e.target.value })} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="act-attendees">{ta("fields.attendees")}</Label>
              <Input id="act-attendees" type="number" min={0} dir="ltr" value={form.attendees_count}
                onChange={(e) => setForm({ ...form, attendees_count: Number(e.target.value) || 0 })} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="act-mode">{ta("fields.attendanceMode")}</Label>
              <NativeSelect id="act-mode" value={form.attendance_mode}
                onChange={(e) => setForm({ ...form, attendance_mode: e.target.value })}>
                <option value="">{t("options.none")}</option>
                {MODES.map((m) => <option key={m} value={m}>{ta(`modes.${m}`)}</option>)}
              </NativeSelect>
            </div>
            <div className="space-y-2">
              <Label htmlFor="act-audience">{ta("fields.targetAudience")}</Label>
              <Input id="act-audience" value={form.target_audience}
                onChange={(e) => setForm({ ...form, target_audience: e.target.value })} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="act-organizer">{ta("fields.organizer")}</Label>
              <Input id="act-organizer" value={form.organizer}
                onChange={(e) => setForm({ ...form, organizer: e.target.value })} />
            </div>
            <div className="space-y-2 md:col-span-2">
              <Label htmlFor="act-desc">{ta("fields.description")}</Label>
              <Textarea id="act-desc" value={form.description}
                onChange={(e) => setForm({ ...form, description: e.target.value })} />
            </div>
            <div className="flex gap-2 md:col-span-2">
              <Button disabled={busy} onClick={save}>{tc("save")}</Button>
              <Button variant="outline" onClick={() => { setForm(null); setEditingId(null); }}>
                {tc("cancel")}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <div className="flex flex-wrap items-center gap-3">
            <div className="relative min-w-56 flex-1">
              <Search className="absolute start-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input className="ps-10" placeholder={ta("searchPlaceholder")} value={filters.q}
                onChange={(e) => { setFilters({ ...filters, q: e.target.value }); setPage(1); }} />
            </div>
            <NativeSelect className="w-auto" value={filters.type}
              onChange={(e) => { setFilters({ ...filters, type: e.target.value }); setPage(1); }}>
              <option value="">{ta("allTypes")}</option>
              {TYPES.map((x) => <option key={x} value={x}>{ta(`types.${x}`)}</option>)}
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
                  <TableHead>{ta("fields.type")}</TableHead>
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
                    canEdit={hydrated && user?.id === a.created_by && (a.status === "pending" || a.status === "returned")}
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
      <TableCell>{ta(`types.${a.activity_type}`)}</TableCell>
      <TableCell className="max-w-xs">
        <span className="block truncate font-medium" title={a.title_ar}>{a.title_ar}</span>
        <span className="block truncate text-xs text-muted-foreground">
          {a.creator_name}{a.presenter ? ` — ${a.presenter}` : ""}
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
      <ActivitiesPage />
    </RequireAuth>
  );
}
