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
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table";
import { NativeSelect, Textarea } from "@/components/portal/form-controls";
import { RequireAuth } from "@/components/portal/require-auth";
import { usePortalErrors } from "@/components/portal/use-portal-errors";
import { formatDate, isStaffRole, type Org } from "@/lib/portal/types";

const TYPES = ["seminar", "workshop", "panel", "lecture", "course", "conference", "other"] as const;
const STATUSES = ["pending", "approved", "returned", "rejected"] as const;
const PAGE_SIZE = 20;

const STATUS_TINT: Record<string, string> = {
  pending: "bg-amber-100 text-amber-800",
  approved: "bg-green-100 text-green-800",
  returned: "bg-orange-100 text-orange-800",
  rejected: "bg-red-100 text-red-800",
};

interface Activity {
  id: string;
  activity_type: string;
  title_ar: string;
  description: string | null;
  presenter: string | null;
  college_id: string | null;
  department_id: string | null;
  college_name: string | null;
  department_name: string | null;
  activity_date: string;
  location: string | null;
  attendees_count: number;
  status: string;
  approval_no: string | null;
  decision_note: string | null;
  created_by: string;
  creator_name: string;
}

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
      })
      .catch(() => setItems([]));
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

  const yearOptions = [0, 1, 2].map((d) => String(new Date().getFullYear() - d));

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold">{ta("title")}</h1>
          <p className="text-sm text-muted-foreground">{ta("subtitle")}</p>
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

      {error && <div className="whitespace-pre-line rounded bg-red-50 p-3 text-sm text-red-600">{error}</div>}

      {form && (
        <Card>
          <CardHeader>
            <CardTitle>{editingId ? t("ui.edit") : ta("new")}</CardTitle>
          </CardHeader>
          <CardContent className="grid gap-4 md:grid-cols-2">
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
            <div className="space-y-2 md:col-span-2">
              <Label htmlFor="act-desc">{ta("fields.description")}</Label>
              <Textarea id="act-desc" value={form.description}
                onChange={(e) => setForm({ ...form, description: e.target.value })} />
            </div>
            <div className="flex gap-2 md:col-span-2">
              <Button disabled={busy} onClick={save}>{tc("save")}</Button>
              <Button variant="outline" onClick={() => { setForm(null); setEditingId(null); }}>
                {t("ui.back")}
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
          {items.length === 0 ? (
            <p className="py-8 text-center text-sm text-muted-foreground">{t("ui.noRequests")}</p>
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
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((a) => (
                  <ActivityRow
                    key={a.id}
                    activity={a}
                    canEdit={hydrated && user?.id === a.created_by && (a.status === "pending" || a.status === "returned")}
                    canDecide={hydrated && staff && (a.status === "pending" || a.status === "returned")}
                    locale={locale}
                    onEdit={() => startEdit(a)}
                    onChanged={load}
                    onError={setError}
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
  canEdit: boolean;
  canDecide: boolean;
  locale: string;
  onEdit: () => void;
  onChanged: () => void;
  onError: (message: string) => void;
}

function ActivityRow({ activity: a, canEdit, canDecide, locale, onEdit, onChanged, onError }: ActivityRowProps) {
  const t = useTranslations("portal");
  const ta = useTranslations("portal.activities");
  const toMessage = usePortalErrors();
  const [open, setOpen] = useState(false);
  const [approvalNo, setApprovalNo] = useState("");
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);

  async function decide(status: string) {
    setBusy(true);
    onError("");
    try {
      await api.post(`/api/v1/activities/${a.id}/decide`, { status, approval_no: approvalNo, note });
      setOpen(false);
      onChanged();
    } catch (err) {
      onError(toMessage(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <TableRow>
        <TableCell>{ta(`types.${a.activity_type}`)}</TableCell>
        <TableCell className="max-w-xs">
          <span className="block truncate font-medium" title={a.title_ar}>{a.title_ar}</span>
          <span className="block truncate text-xs text-muted-foreground">
            {a.creator_name}{a.presenter ? ` — ${a.presenter}` : ""}
          </span>
        </TableCell>
        <TableCell>{a.college_name ?? "—"}</TableCell>
        <TableCell dir="ltr" className="whitespace-nowrap">
          <span className="inline-flex items-center gap-1">
            <CalendarDays className="size-3.5 text-muted-foreground" />
            {formatDate(a.activity_date, locale)}
          </span>
        </TableCell>
        <TableCell className="tabular-nums">{a.attendees_count}</TableCell>
        <TableCell dir="ltr">{a.approval_no ?? "—"}</TableCell>
        <TableCell>
          <div className="flex items-center gap-2">
            <Badge variant="secondary" className={STATUS_TINT[a.status]}>{ta(`statuses.${a.status}`)}</Badge>
            {canEdit && (
              <Button size="xs" variant="outline" onClick={onEdit}>{t("ui.edit")}</Button>
            )}
            {canDecide && (
              <Button size="xs" onClick={() => setOpen((v) => !v)}>{ta("decide")}</Button>
            )}
          </div>
        </TableCell>
      </TableRow>
      {open && (
        <TableRow>
          <TableCell colSpan={7} className="bg-muted/40">
            <div className="flex flex-wrap items-end gap-3 py-1">
              <div className="space-y-1">
                <Label htmlFor={`no-${a.id}`} className="text-xs">{ta("fields.approvalNo")} *</Label>
                <Input id={`no-${a.id}`} dir="ltr" className="w-44" value={approvalNo}
                  onChange={(e) => setApprovalNo(e.target.value)} placeholder={ta("approvalPlaceholder")} />
              </div>
              <div className="min-w-56 flex-1 space-y-1">
                <Label htmlFor={`note-${a.id}`} className="text-xs">{t("ui.noteLabel")}</Label>
                <Input id={`note-${a.id}`} value={note} onChange={(e) => setNote(e.target.value)} />
              </div>
              <div className="flex gap-2">
                <Button size="sm" disabled={busy} onClick={() => decide("approved")}>{ta("approve")}</Button>
                <Button size="sm" variant="outline" disabled={busy} onClick={() => decide("returned")}>
                  {t("transitions.returned")}
                </Button>
                <Button size="sm" variant="destructive" disabled={busy} onClick={() => decide("rejected")}>
                  {t("transitions.rejected")}
                </Button>
              </div>
            </div>
          </TableCell>
        </TableRow>
      )}
    </>
  );
}

export default function Page() {
  return (
    <RequireAuth>
      <ActivitiesPage />
    </RequireAuth>
  );
}
