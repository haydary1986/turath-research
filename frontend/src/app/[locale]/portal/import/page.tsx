"use client";

import { useRef, useState } from "react";
import { useTranslations } from "next-intl";
import { FileSpreadsheet, Upload } from "lucide-react";
import { api } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table";
import { RequireAuth } from "@/components/portal/require-auth";

interface Problem {
  line: number;
  code: string;
}

interface PreviewRow {
  line: number;
  name_ar: string;
  name_en: string;
  email: string;
  title_ar: string;
  college: string;
  department: string;
}

interface ImportReport {
  dry_run: boolean;
  valid_rows: number;
  created: number;
  updated: number;
  users_linked: number;
  colleges_created: number;
  departments_created: number;
  problems: Problem[];
  missing_columns?: string[];
  preview?: PreviewRow[];
}

const COLUMNS = ["name_ar", "name_en", "email", "title_ar", "college", "department", "orcid"] as const;

function ImportPage() {
  const t = useTranslations("portal");
  const ti = useTranslations("portal.import");
  const inputRef = useRef<HTMLInputElement>(null);
  const [file, setFile] = useState<File | null>(null);
  const [report, setReport] = useState<ImportReport | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function run(dryRun: boolean) {
    if (!file) return;
    setError("");
    setBusy(true);
    try {
      const res = await api.upload<ImportReport>(
        `/api/v1/admin/import/researchers?dry_run=${dryRun}`,
        file
      );
      setReport(res.data ?? null);
    } catch (err) {
      const code = err instanceof Error ? err.message : "";
      setError(ti.has(`errors.${code}`) ? ti(`errors.${code}`) : t("errors.generic"));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{ti("title")}</h1>
        <p className="text-sm text-muted-foreground">{ti("intro")}</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{ti("columnsTitle")}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-wrap gap-2 text-sm">
            {COLUMNS.map((c) => (
              <span key={c} className="rounded-lg border px-2 py-1">
                {ti(`columns.${c}`)}
                {(c === "name_ar" || c === "email") && <b className="text-red-600"> *</b>}
              </span>
            ))}
          </div>
          <p className="mt-2 text-xs text-muted-foreground">{ti("columnsHint")}</p>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="space-y-4 pt-4">
          {error && <div className="rounded bg-red-50 p-3 text-sm text-red-600">{error}</div>}
          <input
            ref={inputRef}
            type="file"
            accept=".xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
            className="hidden"
            onChange={(e) => {
              setFile(e.target.files?.[0] ?? null);
              setReport(null);
            }}
          />
          <div className="flex flex-wrap items-center gap-3">
            <Button variant="outline" onClick={() => inputRef.current?.click()}>
              <FileSpreadsheet className="size-4" />
              <span>{ti("chooseFile")}</span>
            </Button>
            {file && <span className="text-sm" dir="ltr">{file.name}</span>}
            <Button disabled={!file || busy} onClick={() => run(true)}>
              {ti("check")}
            </Button>
            {report?.dry_run && !report.missing_columns && report.valid_rows > 0 && (
              <Button disabled={busy} onClick={() => run(false)}>
                <Upload className="size-4" />
                <span>{ti("confirmImport", { count: report.valid_rows })}</span>
              </Button>
            )}
          </div>
        </CardContent>
      </Card>

      {report && (
        <Card>
          <CardHeader>
            <CardTitle>{report.dry_run ? ti("previewTitle") : ti("resultTitle")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {report.missing_columns ? (
              <p className="text-sm text-red-600">
                {ti("missingColumns")}: {report.missing_columns.map((c) => ti(`columns.${c}`)).join("، ")}
              </p>
            ) : (
              <div className="flex flex-wrap gap-4 text-sm">
                <span>{ti("validRows")}: <b>{report.valid_rows}</b></span>
                {!report.dry_run && (
                  <>
                    <span className="text-green-700">{ti("created")}: <b>{report.created}</b></span>
                    <span>{ti("updated")}: <b>{report.updated}</b></span>
                    <span>{ti("usersLinked")}: <b>{report.users_linked}</b></span>
                    <span>{ti("orgsCreated")}: <b>{report.colleges_created + report.departments_created}</b></span>
                  </>
                )}
                <span className={report.problems.length ? "text-red-600" : ""}>
                  {ti("problems")}: <b>{report.problems.length}</b>
                </span>
              </div>
            )}

            {report.problems.length > 0 && (
              <ul className="list-disc ps-5 text-sm text-red-600">
                {report.problems.slice(0, 30).map((p, i) => (
                  <li key={i}>
                    {ti("line")} {p.line}: {ti.has(`errors.${p.code}`) ? ti(`errors.${p.code}`) : p.code}
                  </li>
                ))}
              </ul>
            )}

            {report.preview && report.preview.length > 0 && (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>#</TableHead>
                    <TableHead>{ti("columns.name_ar")}</TableHead>
                    <TableHead>{ti("columns.email")}</TableHead>
                    <TableHead>{ti("columns.title_ar")}</TableHead>
                    <TableHead>{ti("columns.college")}</TableHead>
                    <TableHead>{ti("columns.department")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {report.preview.map((row) => (
                    <TableRow key={row.line}>
                      <TableCell>{row.line}</TableCell>
                      <TableCell>{row.name_ar}</TableCell>
                      <TableCell dir="ltr">{row.email}</TableCell>
                      <TableCell>{row.title_ar || "—"}</TableCell>
                      <TableCell>{row.college || "—"}</TableCell>
                      <TableCell>{row.department || "—"}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}

export default function AdminImportPage() {
  return (
    <RequireAuth>
      <ImportPage />
    </RequireAuth>
  );
}
