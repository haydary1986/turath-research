"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { FileSpreadsheet } from "lucide-react";
import { downloadFile } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { NativeSelect } from "@/components/portal/form-controls";
import { RequireAuth } from "@/components/portal/require-auth";
import { usePortalErrors } from "@/components/portal/use-portal-errors";
import { Alert } from "@/components/ui/alert";

function ReportsPage() {
  const tr = useTranslations("portal.reports");
  const toMessage = usePortalErrors();
  const [year, setYear] = useState(String(new Date().getFullYear()));
  const [period, setPeriod] = useState("year");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const years = [0, 1, 2, 3].map((d) => String(new Date().getFullYear() - d));
  // Quarter → [from, to] within the chosen year (ministry follow-ups are often quarterly).
  const QUARTERS: Record<string, [string, string]> = {
    q1: ["01-01", "03-31"],
    q2: ["04-01", "06-30"],
    q3: ["07-01", "09-30"],
    q4: ["10-01", "12-31"],
  };

  async function download() {
    setError("");
    setBusy(true);
    try {
      let query = `year=${year}`;
      let slug = year;
      if (period !== "year" && QUARTERS[period]) {
        const [from, to] = QUARTERS[period];
        query = `from=${year}-${from}&to=${year}-${to}`;
        slug = `${year}-${from}_${year}-${to}`;
      }
      await downloadFile(`/api/v1/reports/annual.xlsx?${query}`, `turath-report-${slug}.xlsx`);
    } catch (err) {
      setError(toMessage(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{tr("title")}</h1>
        <p className="text-sm text-muted-foreground">{tr("subtitle")}</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <FileSpreadsheet className="size-5 text-primary" />
            {tr("annual")}
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">{tr("annualDesc")}</p>
          {error && <Alert variant="danger">{error}</Alert>}
          <div className="flex items-end gap-3">
            <div className="space-y-2">
              <Label htmlFor="report-year">{tr("year")}</Label>
              <NativeSelect id="report-year" className="w-32" value={year}
                onChange={(e) => setYear(e.target.value)}>
                {years.map((y) => <option key={y} value={y}>{y}</option>)}
              </NativeSelect>
            </div>
            <div className="space-y-2">
              <Label htmlFor="report-period">{tr("period")}</Label>
              <NativeSelect id="report-period" className="w-40" value={period}
                onChange={(e) => setPeriod(e.target.value)}>
                <option value="year">{tr("fullYear")}</option>
                {["q1", "q2", "q3", "q4"].map((q) => (
                  <option key={q} value={q}>{tr(`quarters.${q}`)}</option>
                ))}
              </NativeSelect>
            </div>
            <Button disabled={busy} onClick={download}>
              {busy ? tr("preparing") : tr("download")}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

export default function Page() {
  return (
    <RequireAuth>
      <ReportsPage />
    </RequireAuth>
  );
}
