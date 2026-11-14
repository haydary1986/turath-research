"use client";

import { useLocale, useTranslations } from "next-intl";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { NativeSelect, Textarea } from "@/components/portal/form-controls";
import {
  WORKSHOP_TYPES,
  dateLabelKey,
  isDetailRequired,
  isEventCategory,
  isInstitutional,
  type Activity,
} from "@/lib/portal/activity";
import type { Org } from "@/lib/portal/types";
import { DataOption } from "@/components/i18n/data-option";

const MODES = ["in_person", "online", "hybrid"] as const;
const RESEARCH = ["local_research", "indexed_research", "international_research"];

export interface ActivityFormState {
  category: string;
  activity_type: string;
  title_ar: string;
  description: string;
  presenter: string;
  college_id: string;
  department_id: string;
  activity_date: string;
  category_detail: string;
  item_count: string;
  notes: string;
  attendees_count: number;
  target_audience: string;
  organizer: string;
  attendance_mode: string;
}

/** A blank form for the first category the user may enter, in their own college. */
export function emptyActivityForm(category: string, collegeId: string, departmentId: string): ActivityFormState {
  const institutional = isInstitutional(category);
  return {
    category,
    activity_type: "seminar",
    title_ar: "",
    description: "",
    presenter: "",
    college_id: institutional ? "" : collegeId,
    department_id: institutional ? "" : departmentId,
    activity_date: "",
    category_detail: "",
    item_count: "",
    notes: "",
    attendees_count: 0,
    target_audience: "",
    organizer: "",
    attendance_mode: "",
  };
}

export function activityToForm(a: Activity): ActivityFormState {
  return {
    category: a.category,
    activity_type: a.activity_type,
    title_ar: a.title_ar,
    description: a.description ?? "",
    presenter: a.presenter ?? "",
    college_id: a.college_id ?? "",
    department_id: a.department_id ?? "",
    activity_date: a.activity_date,
    category_detail: a.category_detail ?? a.location ?? "",
    item_count: a.item_count === null ? "" : String(a.item_count),
    notes: a.notes ?? "",
    attendees_count: a.attendees_count,
    target_audience: a.target_audience ?? "",
    organizer: a.organizer ?? "",
    attendance_mode: a.attendance_mode ?? "",
  };
}

/** The request body: event fields only for event sheets, and the venue doubles as the location. */
export function activityPayload(form: ActivityFormState): Record<string, unknown> {
  const event = isEventCategory(form.category);
  const infrastructure = form.category === "infrastructure";
  return {
    category: form.category,
    activity_type: form.category === "workshops" ? form.activity_type : "",
    title_ar: form.title_ar,
    college_id: form.college_id,
    department_id: form.department_id,
    activity_date: form.activity_date,
    category_detail: infrastructure ? "" : form.category_detail,
    item_count: infrastructure && form.item_count !== "" ? Number(form.item_count) : null,
    notes: form.notes,
    location: event ? form.category_detail : "",
    description: event ? form.description : "",
    presenter: event ? form.presenter : "",
    attendees_count: event ? form.attendees_count : 0,
    target_audience: event ? form.target_audience : "",
    organizer: event ? form.organizer : "",
    attendance_mode: event ? form.attendance_mode : "",
  };
}

interface ActivityFormProps {
  form: ActivityFormState;
  editing: boolean;
  editingReturned: boolean;
  categories: readonly string[];
  colleges: Org[];
  departments: Org[];
  busy: boolean;
  onChange: (form: ActivityFormState) => void;
  onSave: () => void;
  onCancel: () => void;
}

export function ActivityForm({
  form, editing, editingReturned, categories, colleges, departments, busy, onChange, onSave, onCancel,
}: ActivityFormProps) {
  const t = useTranslations("portal");
  const ta = useTranslations("portal.activities");
  const tc = useTranslations("common");
  const locale = useLocale();
  const set = (change: Partial<ActivityFormState>) => onChange({ ...form, ...change });
  const institutional = isInstitutional(form.category);
  const event = isEventCategory(form.category);
  const options = categories.includes(form.category) ? categories : [form.category, ...categories];

  return (
    <Card>
      <CardHeader>
        <CardTitle>{editing ? t("ui.edit") : ta("new")}</CardTitle>
      </CardHeader>
      <CardContent className="grid gap-4 md:grid-cols-2">
        {editingReturned && <Alert variant="info" className="md:col-span-2">{ta("resubmitHint")}</Alert>}
        <div className="space-y-2 md:col-span-2">
          <Label htmlFor="act-category">{ta("fields2.category")} *</Label>
          <NativeSelect id="act-category" value={form.category}
            onChange={(e) => {
              const category = e.target.value;
              set(isInstitutional(category) && !institutional
                ? { category, college_id: "", department_id: "" }
                : { category });
            }}>
            {options.map((c) => <option key={c} value={c}>{ta(`categories.${c}`)}</option>)}
          </NativeSelect>
        </div>
        {institutional && <Alert variant="info" className="md:col-span-2">{ta("institutionalHint")}</Alert>}
        {RESEARCH.includes(form.category) && <Alert variant="info" className="md:col-span-2">{ta("researchHint")}</Alert>}
        <div className="space-y-2 md:col-span-2">
          <Label htmlFor="act-title">{ta("fields.title")} *</Label>
          <Input id="act-title" dir="rtl" value={form.title_ar} onChange={(e) => set({ title_ar: e.target.value })} />
        </div>
        {form.category === "workshops" && (
          <div className="space-y-2">
            <Label htmlFor="act-type">{ta("fields2.subType")} *</Label>
            <NativeSelect id="act-type" value={form.activity_type} onChange={(e) => set({ activity_type: e.target.value })}>
              {WORKSHOP_TYPES.map((x) => <option key={x} value={x}>{ta(`types.${x}`)}</option>)}
            </NativeSelect>
          </div>
        )}
        <div className="space-y-2">
          <Label htmlFor="act-date">{ta(`dateLabels.${dateLabelKey(form.category)}`)} *</Label>
          <Input id="act-date" type="date" dir="ltr" value={form.activity_date}
            onChange={(e) => set({ activity_date: e.target.value })} />
        </div>
        {form.category === "infrastructure" ? (
          <div className="space-y-2">
            <Label htmlFor="act-count">{ta("fields2.itemCount")} *</Label>
            <Input id="act-count" type="number" min={1} dir="ltr" value={form.item_count}
              onChange={(e) => set({ item_count: e.target.value })} />
          </div>
        ) : (
          <div className="space-y-2">
            <Label htmlFor="act-detail">
              {ta(`detailLabels.${form.category}`)}{isDetailRequired(form.category) ? " *" : ""}
            </Label>
            <Input id="act-detail" value={form.category_detail} onChange={(e) => set({ category_detail: e.target.value })} />
          </div>
        )}
        <div className="space-y-2">
          <Label htmlFor="act-college">{t("fields.college")}{institutional ? "" : " *"}</Label>
          <NativeSelect id="act-college" value={form.college_id}
            onChange={(e) => set({ college_id: e.target.value, department_id: "" })}>
            <option value="">{institutional ? ta("universityLevel") : t("options.none")}</option>
            {colleges.map((c) => <DataOption key={c.id} value={c.id} ar={c.name_ar} en={c.name_en} locale={locale} />)}
          </NativeSelect>
        </div>
        <div className="space-y-2">
          <Label htmlFor="act-dept">{t("fields.department")}</Label>
          <NativeSelect id="act-dept" value={form.department_id} disabled={!form.college_id && institutional}
            onChange={(e) => set({ department_id: e.target.value })}>
            <option value="">{t("options.none")}</option>
            {departments.filter((d) => !form.college_id || d.college_id === form.college_id)
              .map((d) => <DataOption key={d.id} value={d.id} ar={d.name_ar} en={d.name_en} locale={locale} />)}
          </NativeSelect>
        </div>
        {event && <EventFields form={form} set={set} />}
        <div className="space-y-2 md:col-span-2">
          <Label htmlFor="act-notes">{ta("fields2.notes")}</Label>
          <Textarea id="act-notes" value={form.notes} onChange={(e) => set({ notes: e.target.value })} />
        </div>
        <div className="flex gap-2 md:col-span-2">
          <Button disabled={busy} onClick={onSave}>{tc("save")}</Button>
          <Button variant="outline" onClick={onCancel}>{tc("cancel")}</Button>
        </div>
      </CardContent>
    </Card>
  );
}

interface EventFieldsProps {
  form: ActivityFormState;
  set: (change: Partial<ActivityFormState>) => void;
}

/** Presenter, attendance and audience: only the event sheets (workshops, courses, conferences, community service). */
function EventFields({ form, set }: EventFieldsProps) {
  const t = useTranslations("portal");
  const ta = useTranslations("portal.activities");
  return (
    <>
      <div className="space-y-2">
        <Label htmlFor="act-presenter">{ta("fields.presenter")}</Label>
        <Input id="act-presenter" value={form.presenter} onChange={(e) => set({ presenter: e.target.value })} />
      </div>
      <div className="space-y-2">
        <Label htmlFor="act-attendees">{ta("fields.attendees")}</Label>
        <Input id="act-attendees" type="number" min={0} dir="ltr" value={form.attendees_count}
          onChange={(e) => set({ attendees_count: Number(e.target.value) || 0 })} />
      </div>
      <div className="space-y-2">
        <Label htmlFor="act-mode">{ta("fields.attendanceMode")}</Label>
        <NativeSelect id="act-mode" value={form.attendance_mode} onChange={(e) => set({ attendance_mode: e.target.value })}>
          <option value="">{t("options.none")}</option>
          {MODES.map((m) => <option key={m} value={m}>{ta(`modes.${m}`)}</option>)}
        </NativeSelect>
      </div>
      <div className="space-y-2">
        <Label htmlFor="act-audience">{ta("fields.targetAudience")}</Label>
        <Input id="act-audience" value={form.target_audience} onChange={(e) => set({ target_audience: e.target.value })} />
      </div>
      <div className="space-y-2">
        <Label htmlFor="act-organizer">{ta("fields.organizer")}</Label>
        <Input id="act-organizer" value={form.organizer} onChange={(e) => set({ organizer: e.target.value })} />
      </div>
      <div className="space-y-2 md:col-span-2">
        <Label htmlFor="act-desc">{ta("fields.description")}</Label>
        <Textarea id="act-desc" value={form.description} onChange={(e) => set({ description: e.target.value })} />
      </div>
    </>
  );
}
