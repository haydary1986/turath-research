"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { useRouter } from "@/i18n/routing";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { fieldsFor, type FieldDef } from "@/lib/portal/fields";
import { requestTitle, type Org, type PortalRequest, type RequestKind } from "@/lib/portal/types";
import { NativeSelect, Textarea } from "./form-controls";
import { usePortalErrors } from "./use-portal-errors";

interface RequestFormProps {
  kind: RequestKind;
  /** Present when editing a draft or returned request. */
  existing?: PortalRequest;
}

interface FormState {
  title_ar: string;
  title_en: string;
  doi: string;
  college_id: string;
  department_id: string;
  related_request_id: string;
  details: Record<string, string>;
}

function initialState(
  existing: PortalRequest | undefined,
  user: { college_id?: string | null; department_id?: string | null } | null
): FormState {
  return {
    title_ar: existing?.title_ar ?? "",
    title_en: existing?.title_en ?? "",
    doi: existing?.doi ?? "",
    college_id: existing?.college_id ?? user?.college_id ?? "",
    department_id: existing?.department_id ?? user?.department_id ?? "",
    related_request_id: existing?.related_request_id ?? "",
    details: { ...(existing?.details ?? {}) },
  };
}

export function RequestForm({ kind, existing }: RequestFormProps) {
  const t = useTranslations("portal");
  const toMessage = usePortalErrors();
  const router = useRouter();
  const { user } = useAuth();

  const [form, setForm] = useState<FormState>(() => initialState(existing, user));
  const [colleges, setColleges] = useState<Org[]>([]);
  const [departments, setDepartments] = useState<Org[]>([]);
  const [verifications, setVerifications] = useState<PortalRequest[]>([]);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;
    Promise.all([
      api.get<Org[]>("/api/v1/colleges"),
      api.get<Org[]>("/api/v1/departments"),
    ])
      .then(([c, d]) => {
        if (!active) return;
        setColleges(c.data ?? []);
        setDepartments(d.data ?? []);
      })
      .catch(() => undefined);
    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    if (kind !== "appreciation_letter") return;
    let active = true;
    api
      .get<PortalRequest[]>(
        "/api/v1/portal/requests?mine=true&kind=journal_verification&status=completed&limit=50"
      )
      .then((res) => active && setVerifications(res.data ?? []))
      .catch(() => undefined);
    return () => {
      active = false;
    };
  }, [kind]);

  function setDetail(key: string, value: string) {
    setForm((prev) => ({ ...prev, details: { ...prev.details, [key]: value } }));
  }

  function selectVerification(id: string) {
    const source = verifications.find((v) => v.id === id);
    setForm((prev) =>
      source
        ? {
            ...prev,
            related_request_id: id,
            title_ar: prev.title_ar || (source.title_ar ?? ""),
            title_en: prev.title_en || (source.title_en ?? ""),
            doi: prev.doi || (source.doi ?? ""),
            college_id: prev.college_id || (source.college_id ?? ""),
            department_id: prev.department_id || (source.department_id ?? ""),
            details: { ...source.details, ...prev.details },
          }
        : { ...prev, related_request_id: "" }
    );
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setSaving(true);
    try {
      const payload = { ...form, kind };
      const res = existing
        ? await api.put<PortalRequest>(`/api/v1/portal/requests/${existing.id}`, payload)
        : await api.post<PortalRequest>("/api/v1/portal/requests", payload);
      if (res.data) router.push(`/portal/requests/${res.data.id}`);
    } catch (err) {
      setError(toMessage(err));
    } finally {
      setSaving(false);
    }
  }

  const visibleDepartments = departments.filter(
    (d) => !form.college_id || d.college_id === form.college_id
  );

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {error && (
        <div className="whitespace-pre-line rounded bg-red-50 p-3 text-sm text-red-600">{error}</div>
      )}

      {kind === "appreciation_letter" && (
        <Card>
          <CardHeader>
            <CardTitle>{t("fields.related_request")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            <NativeSelect
              value={form.related_request_id}
              onChange={(e) => selectVerification(e.target.value)}
            >
              <option value="">{t("ui.noRelated")}</option>
              {verifications.map((v) => (
                <option key={v.id} value={v.id}>
                  {v.request_no} — {requestTitle(v)}
                </option>
              ))}
            </NativeSelect>
            <p className="text-xs text-muted-foreground">{t("ui.reuseHint")}</p>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle>{t("ui.step1")}</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-4 md:grid-cols-2">
          <div className="space-y-2 md:col-span-2">
            <Label htmlFor="title_ar">{t("fields.title_ar")}{kind === "research_plan" && " *"}</Label>
            <Input
              id="title_ar"
              dir="rtl"
              value={form.title_ar}
              onChange={(e) => setForm({ ...form, title_ar: e.target.value })}
            />
          </div>
          <div className="space-y-2 md:col-span-2">
            <Label htmlFor="title_en">{t("fields.title_en")} *</Label>
            <Input
              id="title_en"
              dir="ltr"
              value={form.title_en}
              onChange={(e) => setForm({ ...form, title_en: e.target.value })}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="doi">{t("fields.doi")}</Label>
            <Input
              id="doi"
              dir="ltr"
              value={form.doi}
              onChange={(e) => setForm({ ...form, doi: e.target.value })}
            />
            <p className="text-xs text-muted-foreground">{t("ui.doiHint")}</p>
          </div>
          <div />
          <div className="space-y-2">
            <Label htmlFor="college">{t("fields.college")}</Label>
            <NativeSelect
              id="college"
              value={form.college_id}
              onChange={(e) => setForm({ ...form, college_id: e.target.value, department_id: "" })}
            >
              <option value="">{t("options.none")}</option>
              {colleges.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name_ar}
                </option>
              ))}
            </NativeSelect>
          </div>
          <div className="space-y-2">
            <Label htmlFor="department">{t("fields.department")}</Label>
            <NativeSelect
              id="department"
              value={form.department_id}
              onChange={(e) => setForm({ ...form, department_id: e.target.value })}
            >
              <option value="">{t("options.none")}</option>
              {visibleDepartments.map((d) => (
                <option key={d.id} value={d.id}>
                  {d.name_ar}
                </option>
              ))}
            </NativeSelect>
          </div>

          {fieldsFor(kind).map((field) => (
            <DetailField
              key={field.key}
              field={field}
              value={form.details[field.key] ?? ""}
              onChange={(value) => setDetail(field.key, value)}
            />
          ))}
        </CardContent>
      </Card>

      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={() => router.back()}>
          {t("ui.back")}
        </Button>
        <Button type="submit" disabled={saving}>
          {saving ? t("ui.saving") : existing ? t("ui.saveChanges") : t("ui.saveContinue")}
        </Button>
      </div>
    </form>
  );
}

interface DetailFieldProps {
  field: FieldDef;
  value: string;
  onChange: (value: string) => void;
}

function DetailField({ field, value, onChange }: DetailFieldProps) {
  const t = useTranslations("portal");
  const id = `detail-${field.key}`;
  const wide = field.type === "textarea" || field.type === "url";
  const label = `${t(`fields.${field.key}`)}${field.required ? " *" : ""}`;

  return (
    <div className={`space-y-2 ${wide ? "md:col-span-2" : ""}`}>
      <Label htmlFor={id}>{label}</Label>
      {field.type === "select" ? (
        <NativeSelect id={id} value={value} onChange={(e) => onChange(e.target.value)}>
          <option value="">{t("options.none")}</option>
          {field.options?.map((option) => (
            <option key={option} value={option}>
              {t.has(`options.${option}`) ? t(`options.${option}`) : option}
            </option>
          ))}
        </NativeSelect>
      ) : field.type === "textarea" ? (
        <Textarea id={id} value={value} onChange={(e) => onChange(e.target.value)} />
      ) : (
        <Input
          id={id}
          type={field.type === "date" ? "date" : "text"}
          dir={field.ltr || field.type === "date" ? "ltr" : undefined}
          value={value}
          onChange={(e) => onChange(e.target.value)}
        />
      )}
    </div>
  );
}
