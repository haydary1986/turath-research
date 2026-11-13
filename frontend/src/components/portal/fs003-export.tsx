"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { FileSpreadsheet } from "lucide-react";
import { downloadFile } from "@/lib/api";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { NativeSelect } from "@/components/portal/form-controls";
import { fs003Filename } from "@/lib/portal/activity";
import type { Org } from "@/lib/portal/types";

/** The current Baghdad month as [first day, last day] in YYYY-MM-DD. */
function currentMonth(): [string, string] {
  const today = new Intl.DateTimeFormat("en-CA", { timeZone: "Asia/Baghdad" }).format(new Date());
  const [year, month] = today.split("-").map(Number);
  const last = new Date(Date.UTC(year, month, 0)).getUTCDate();
  const mm = String(month).padStart(2, "0");
  return [`${year}-${mm}-01`, `${year}-${mm}-${String(last).padStart(2, "0")}`];
}

interface FS003ExportProps {
  colleges: Org[];
  onClose: () => void;
}

/** Downloads the ministry form FS 003 for a period, optionally for one college. */
export function FS003Export({ colleges, onClose }: FS003ExportProps) {
  const t = useTranslations("portal");
  const tf = useTranslations("portal.activities.fs003");
  const errors = useTranslations("portal.errors");
  const tc = useTranslations("common");
  const [[from, to], setRange] = useState<[string, string]>(currentMonth);
  const [collegeId, setCollegeId] = useState("");
  const [coordinator, setCoordinator] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function download() {
    setError("");
    if (!from || !to || to < from) {
      setError(`${errors("validationFailed")}\n${t("fields.to")}: ${errors("invalid")}`);
      return;
    }
    setBusy(true);
    try {
      const params = new URLSearchParams({ from, to });
      if (collegeId) params.set("college_id", collegeId);
      if (coordinator.trim()) params.set("coordinator", coordinator.trim());
      await downloadFile(`/api/v1/activities/fs003.xlsx?${params}`, fs003Filename(from, to));
    } catch (err) {
      const message = err instanceof Error ? err.message : "";
      setError(message.endsWith("403") ? errors("fs003_forbidden") : errors("generic"));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <FileSpreadsheet className="size-5 text-primary" />
          {tf("title")}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-sm text-muted-foreground">{tf("hint")}</p>
        {error && <Alert variant="danger" className="whitespace-pre-line">{error}</Alert>}
        <div className="grid gap-4 md:grid-cols-4">
          <div className="space-y-2">
            <Label htmlFor="fs-from">{t("fields.from")}</Label>
            <Input id="fs-from" type="date" dir="ltr" value={from} onChange={(e) => setRange([e.target.value, to])} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="fs-to">{t("fields.to")}</Label>
            <Input id="fs-to" type="date" dir="ltr" value={to} onChange={(e) => setRange([from, e.target.value])} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="fs-college">{t("fields.college")}</Label>
            <NativeSelect id="fs-college" value={collegeId} onChange={(e) => setCollegeId(e.target.value)}>
              <option value="">{tf("allColleges")}</option>
              {colleges.map((c) => <option key={c.id} value={c.id}>{c.name_ar}</option>)}
            </NativeSelect>
          </div>
          <div className="space-y-2">
            <Label htmlFor="fs-coordinator">{t("fields.coordinator")}</Label>
            <Input id="fs-coordinator" value={coordinator} maxLength={200} placeholder={tf("coordinatorPlaceholder")}
              onChange={(e) => setCoordinator(e.target.value)} />
          </div>
        </div>
        <div className="flex gap-2">
          <Button disabled={busy} onClick={download}>{busy ? tf("preparing") : tf("download")}</Button>
          <Button variant="outline" onClick={onClose}>{tc("cancel")}</Button>
        </div>
      </CardContent>
    </Card>
  );
}
