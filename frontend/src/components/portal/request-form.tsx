"use client";

import { useEffect, useRef, useState } from "react";
import { useTranslations } from "next-intl";
import { useRouter } from "@/i18n/routing";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { fieldsFor, quartileApplies, withoutStaleQuartile, type FieldDef } from "@/lib/portal/fields";
import { requestTitle, type JournalCheck, type Org, type PortalRequest, type RequestKind } from "@/lib/portal/types";
import { JournalEvidencePanel } from "./journal-evidence-panel";
import { useJournalCheck } from "./use-journal-check";
import { toLatinDigits } from "@/lib/portal/journal-query";

const ISSN_FIELDS = new Set(["issn", "issn_online"]);
import { NativeSelect, Textarea } from "./form-controls";
import { usePortalErrors } from "./use-portal-errors";
import { Alert } from "@/components/ui/alert";

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

function journalSuggestions(result: JournalCheck): { field: string; value: string }[] {
  const pairs: [string, string | undefined][] = [
    ["journal_name", result.canonical_name],
    ["publisher", result.publisher],
    ["issn_online", result.issn_online],
  ];
  return pairs.flatMap(([field, value]) => (value ? [{ field, value }] : []));
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

  const hasJournal = kind !== "research_plan";
  const check = useJournalCheck(
    hasJournal
      ? {
          issn: form.details.issn,
          issn_online: form.details.issn_online,
          name: form.details.journal_name,
          publisher: form.details.publisher,
          url: form.details.journal_url,
        }
      : {}
  );
  const checkResult = check.result;

  // Fills each empty field once per suggested value, so clearing a field keeps it clear.
  const autoFilled = useRef(new Set<string>());
  const [didAutoFill, setDidAutoFill] = useState(false);
  const details = form.details;
  useEffect(() => {
    if (!checkResult) return;
    const fill = journalSuggestions(checkResult).filter(
      ({ field, value }) => !details[field]?.trim() && !autoFilled.current.has(`${field}:${value}`)
    );
    if (fill.length === 0) return;
    fill.forEach(({ field, value }) => autoFilled.current.add(`${field}:${value}`));
    setDidAutoFill(true);
    setForm((prev) => ({
      ...prev,
      details: { ...prev.details, ...Object.fromEntries(fill.map(({ field, value }) => [field, value])) },
    }));
  }, [checkResult, details]);

  const offers = checkResult
    ? journalSuggestions(checkResult).filter(({ field, value }) => {
        const current = form.details[field]?.trim() ?? "";
        return current !== "" && current.toLowerCase() !== value.toLowerCase();
      })
    : [];

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
    setForm((prev) => ({
      ...prev,
      details: withoutStaleQuartile({ ...prev.details, [key]: ISSN_FIELDS.has(key) ? toLatinDigits(value) : value }),
    }));
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
      const payload = { ...form, kind, details: withoutStaleQuartile(form.details) };
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
        <Alert variant="danger">{error}</Alert>
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

      {hasJournal && (
        <JournalEvidencePanel result={check.result} loading={check.loading} failed={check.failed} onRetry={check.retry}>
          {didAutoFill && <p className="text-xs text-muted-foreground">{t("journals.check.suggest.autoFilled")}</p>}
          {offers.length > 0 && (
            <div className="space-y-2">
              <p className="font-medium">{t("journals.check.suggest.title")}</p>
              <ul className="space-y-2">
                {offers.map(({ field, value }) => (
                  <li key={field} className="flex flex-wrap items-center justify-between gap-2">
                    <span className="min-w-0">
                      <span className="text-muted-foreground">{t(`fields.${field}`)}: </span>
                      <span dir="auto">{value}</span>
                    </span>
                    <Button type="button" variant="outline" size="sm" onClick={() => setDetail(field, value)}>
                      {t("journals.check.suggest.use")}
                    </Button>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </JournalEvidencePanel>
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
          {kind !== "research_plan" && (
            <>
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
            </>
          )}
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

          {fieldsFor(kind)
            .filter((field) => field.key !== "quartile" || quartileApplies(form.details.index_db))
            .map((field) => (
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
