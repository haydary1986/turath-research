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

function ReportsPage() {
  const tr = useTranslations("portal.reports");
  const toMessage = usePortalErrors();
  const [year, setYear] = useState(String(new Date().getFullYear()));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const years = [0, 1, 2, 3].map((d) => String(new Date().getFullYear() - d));

  async function download() {
    setError("");
    setBusy(true);
    try {
      await downloadFile(`/api/v1/reports/annual.xlsx?year=${year}`, `turath-report-${year}.xlsx`);
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
          {error && <div className="rounded bg-red-50 p-3 text-sm text-red-600">{error}</div>}
          <div className="flex items-end gap-3">
            <div className="space-y-2">
              <Label htmlFor="report-year">{tr("year")}</Label>
              <NativeSelect id="report-year" className="w-32" value={year}
                onChange={(e) => setYear(e.target.value)}>
                {years.map((y) => <option key={y} value={y}>{y}</option>)}
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
