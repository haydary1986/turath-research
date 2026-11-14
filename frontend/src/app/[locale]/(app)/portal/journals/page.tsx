"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import { FileUp, Plus, Search, SearchCheck, ShieldAlert, ShieldCheck, ShieldX, Trash2 } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { Link } from "@/i18n/routing";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
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
import { Alert } from "@/components/ui/alert";
import { FAMILY_TINT, SCREENING_FAMILY } from "@/lib/portal/status";
import { isStaffRole } from "@/lib/portal/types";
import { Bidi, Ltr } from "@/components/i18n/bidi";
import { bidiProps } from "@/components/i18n/bidi";

const CLASSES = ["accredited", "predatory", "suspended"] as const;
type Classification = (typeof CLASSES)[number];

const CLASS_STYLE: Record<Classification, { badge: string; icon: LucideIcon }> = {
  accredited: { badge: FAMILY_TINT[SCREENING_FAMILY.accredited], icon: ShieldCheck },
  predatory: { badge: FAMILY_TINT[SCREENING_FAMILY.predatory], icon: ShieldX },
  suspended: { badge: FAMILY_TINT[SCREENING_FAMILY.suspended], icon: ShieldAlert },
};

interface Entry {
  id: string;
  issn: string | null;
  name: string;
  publisher: string | null;
  classification: Classification;
  note: string | null;
  source: string | null;
}

interface ImportReport {
  dry_run: boolean;
  valid_rows: number;
  created: number;
  updated: number;
  problems: { line: number; code: string }[];
  missing_columns?: string[];
}

interface FormState {
  id: string | null;
  issn: string;
  name: string;
  publisher: string;
  classification: Classification;
  note: string;
  source: string;
}

const EMPTY: FormState = {
  id: null, issn: "", name: "", publisher: "",
  classification: "accredited", note: "", source: "",
};

function JournalsPage() {
  const t = useTranslations("portal");
  const tj = useTranslations("portal.journals");
  const tc = useTranslations("common");
  const toMessage = usePortalErrors();
  const locale = useLocale();
  const [items, setItems] = useState<Entry[]>([]);
  const [loading, setLoading] = useState(true);
  const [importReport, setImportReport] = useState<ImportReport | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const [q, setQ] = useState("");
  const [classFilter, setClassFilter] = useState("");
  const [form, setForm] = useState<FormState | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const load = useCallback(() => {
    const params = new URLSearchParams({ limit: "100" });
    if (q.trim()) params.set("q", q.trim());
    if (classFilter) params.set("classification", classFilter);
    api
      .get<Entry[]>(`/api/v1/screening?${params}`)
      .then((res) => setItems(res.data ?? []))
      .catch((err) => setError(toMessage(err)))
      .finally(() => setLoading(false));
  }, [q, classFilter, toMessage]);

  useEffect(() => {
    const timer = setTimeout(load, 250);
    return () => clearTimeout(timer);
  }, [load]);

  async function save() {
    if (!form) return;
    setError("");
    setBusy(true);
    try {
      const { id, ...payload } = form;
      if (id) {
        await api.put(`/api/v1/screening/${id}`, payload);
      } else {
        await api.post("/api/v1/screening", payload);
      }
      setForm(null);
      load();
    } catch (err) {
      setError(toMessage(err));
    } finally {
      setBusy(false);
    }
  }

  async function remove(id: string) {
    if (!window.confirm(tj("deleteConfirm"))) return;
    setError("");
    try {
      await api.delete(`/api/v1/screening/${id}`);
      load();
    } catch (err) {
      setError(toMessage(err));
    }
  }

  async function importFile(file: File) {
    setError("");
    setBusy(true);
    setImportReport(null);
    try {
      const res = await api.upload<ImportReport>("/api/v1/screening/import", file);
      setImportReport(res.data ?? null);
      load();
    } catch (err) {
      const failedLine = err instanceof Error ? /^import_failed_line:(\d+)$/.exec(err.message) : null;
      setError(failedLine ? tj("importFailedLine", { line: failedLine[1] }) : toMessage(err));
    } finally {
      setBusy(false);
      if (inputRef.current) inputRef.current.value = "";
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold">{tj("manual.title")}</h1>
          <p className="text-sm text-muted-foreground">{tj("manual.description")}</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" nativeButton={false} render={<Link href="/portal/journal-check" />}>
            <SearchCheck className="size-4" />
            <span>{tj("check.title")}</span>
          </Button>
          <input
            ref={inputRef}
            type="file"
            accept=".xlsx"
            className="hidden"
            onChange={(e) => e.target.files?.[0] && importFile(e.target.files[0])}
          />
          <Button variant="outline" disabled={busy} onClick={() => inputRef.current?.click()}>
            <FileUp className="size-4" />
            <span>{tj("import")}</span>
          </Button>
          <Button variant="outline" onClick={() => setForm(form ? null : { ...EMPTY })}>
            <Plus className="size-4" />
            <span>{tj("new")}</span>
          </Button>
        </div>
      </div>

      {error && <Alert variant="danger">{error}</Alert>}
      {importReport?.missing_columns?.length ? (
        <Alert variant="danger">
          {tj("importMissingColumns", {
            columns: importReport.missing_columns
              .map((column) => (tj.has(`importColumns.${column}`) ? tj(`importColumns.${column}`) : column))
              .join(locale === "ar" ? "، " : ", "),
          })}
        </Alert>
      ) : importReport && (
        <Alert variant={importReport.problems.length ? "warning" : "success"}>
          {tj("importResult", {
            created: importReport.created,
            updated: importReport.updated,
            problems: importReport.problems.length,
          })}
        </Alert>
      )}

      {form && (
        <Card>
          <CardHeader>
            <CardTitle>{form.id ? t("ui.edit") : tj("new")}</CardTitle>
          </CardHeader>
          <CardContent className="grid gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="js-name">{t("fields.journal_name")} *</Label>
              <Input id="js-name" dir="auto" value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="js-issn">ISSN</Label>
              <Input id="js-issn" dir="ltr" placeholder="1234-567X" value={form.issn}
                onChange={(e) => setForm({ ...form, issn: e.target.value })} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="js-class">{tj("classification")} *</Label>
              <NativeSelect id="js-class" value={form.classification}
                onChange={(e) => setForm({ ...form, classification: e.target.value as Classification })}>
                {CLASSES.map((c) => <option key={c} value={c}>{tj(`classes.${c}`)}</option>)}
              </NativeSelect>
            </div>
            <div className="space-y-2">
              <Label htmlFor="js-publisher">{t("fields.publisher")}</Label>
              <Input id="js-publisher" value={form.publisher}
                onChange={(e) => setForm({ ...form, publisher: e.target.value })} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="js-source">{tj("source")}</Label>
              <Input id="js-source" value={form.source} placeholder={tj("sourceHint")}
                onChange={(e) => setForm({ ...form, source: e.target.value })} />
            </div>
            <div className="space-y-2 md:col-span-2">
              <Label htmlFor="js-note">{t("fields.notes")}</Label>
              <Textarea id="js-note" value={form.note}
                onChange={(e) => setForm({ ...form, note: e.target.value })} />
            </div>
            <div className="flex gap-2 md:col-span-2">
              <Button disabled={busy} onClick={save}>{tj("save")}</Button>
              <Button variant="outline" onClick={() => setForm(null)}>{tc("cancel")}</Button>
            </div>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <div className="flex flex-wrap items-center gap-3">
            <div className="relative min-w-56 flex-1">
              <Search className="absolute start-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input className="ps-10" placeholder={tj("searchPlaceholder")} value={q}
                onChange={(e) => setQ(e.target.value)} />
            </div>
            <NativeSelect className="w-auto" value={classFilter}
              onChange={(e) => setClassFilter(e.target.value)}>
              <option value="">{tj("allClasses")}</option>
              {CLASSES.map((c) => <option key={c} value={c}>{tj(`classes.${c}`)}</option>)}
            </NativeSelect>
          </div>
        </CardHeader>
        <CardContent>
          {loading ? (
            <p className="py-8 text-center text-sm text-muted-foreground">{t("ui.loading")}</p>
          ) : items.length === 0 ? (
            <p className="py-8 text-center text-sm text-muted-foreground">{tj("empty")}</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("fields.journal_name")}</TableHead>
                  <TableHead>ISSN</TableHead>
                  <TableHead>{t("fields.publisher")}</TableHead>
                  <TableHead>{tj("classification")}</TableHead>
                  <TableHead>{tj("source")}</TableHead>
                  <TableHead />
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((entry) => {
                  const Icon = CLASS_STYLE[entry.classification].icon;
                  return (
                    <TableRow key={entry.id}>
                      <TableCell className="max-w-xs">
                        <button type="button" className="block max-w-full truncate text-start font-medium hover:underline"
                          title={entry.name}
                          {...bidiProps(entry.name)}
                          onClick={() => setForm({
                            id: entry.id, issn: entry.issn ?? "", name: entry.name,
                            publisher: entry.publisher ?? "", classification: entry.classification,
                            note: entry.note ?? "", source: entry.source ?? "",
                          })}>
                          <Bidi text={entry.name} />
                        </button>
                        <Bidi block text={entry.note} title={entry.note ?? undefined} className="truncate text-xs text-muted-foreground" />
                      </TableCell>
                      <TableCell><Ltr>{entry.issn ?? "—"}</Ltr></TableCell>
                      <TableCell><Bidi block text={entry.publisher} fallback="—" /></TableCell>
                      <TableCell>
                        <Badge variant="secondary" className={CLASS_STYLE[entry.classification].badge}>
                          <Icon className="size-3" />
                          {tj(`classes.${entry.classification}`)}
                        </Badge>
                      </TableCell>
                      <TableCell className="max-w-40"><Bidi block text={entry.source} title={entry.source ?? ""} className="truncate" fallback="—" /></TableCell>
                      <TableCell>
                        <Button variant="ghost" size="icon-sm" aria-label={t("ui.remove")}
                          onClick={() => remove(entry.id)}>
                          <Trash2 className="size-3.5 text-destructive" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

/** The list is staff-only on the API, so others are sent to the check, not to a failing screen. */
function StaffOnly() {
  const tj = useTranslations("portal.journals");
  const role = useAuth((state) => state.user?.role);
  if (isStaffRole(role)) return <JournalsPage />;
  return (
    <Card className="mx-auto max-w-md">
      <CardContent className="space-y-4 py-8 text-center">
        <p>{tj("staffOnly")}</p>
        <Button nativeButton={false} render={<Link href="/portal/journal-check" />}>{tj("check.title")}</Button>
      </CardContent>
    </Card>
  );
}

export default function Page() {
  return (
    <RequireAuth>
      <StaffOnly />
    </RequireAuth>
  );
}
