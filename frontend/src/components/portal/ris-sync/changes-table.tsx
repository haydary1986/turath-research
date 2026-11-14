"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { api } from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Link } from "@/i18n/routing";
import { Bidi } from "@/components/i18n/bidi";
import { NativeSelect } from "@/components/portal/form-controls";
import { RIS_SYNC_API, RIS_SYNC_FIELDS, type RISSyncChange, type RISSyncRun } from "@/lib/portal/ris-sync";
import { ValueCell } from "./value-cell";

interface ChangesTableProps {
  run: RISSyncRun;
  departments: Map<string, string>;
}

/** What one run changed (or would change), filterable by field and direction. */
export function ChangesTable({ run, departments }: ChangesTableProps) {
  const t = useTranslations("portal.risSync");
  const [result, setResult] = useState<{ runId: string; items: RISSyncChange[] }>({ runId: "", items: [] });
  const [field, setField] = useState("");
  const [action, setAction] = useState("");

  useEffect(() => {
    let active = true;
    api.get<RISSyncChange[]>(RIS_SYNC_API.changes(run.id))
      .then((res) => active && setResult({ runId: run.id, items: res.data ?? [] }))
      .catch(() => active && setResult({ runId: run.id, items: [] }));
    return () => {
      active = false;
    };
  }, [run.id]);

  const loading = result.runId !== run.id;
  const items = result.items.filter((c) => (!field || c.field === field) && (!action || c.action === action));

  return (
    <Card>
      <CardHeader className="flex-row flex-wrap items-end justify-between gap-3">
        <CardTitle>{t("changesTitle", { mode: t(`modes.${run.mode}`) })}</CardTitle>
        <div className="flex flex-wrap gap-2">
          <NativeSelect aria-label={t("field")} value={field} onChange={(e) => setField(e.target.value)} className="h-9">
            <option value="">{t("allFields")}</option>
            {RIS_SYNC_FIELDS.map((f) => <option key={f} value={f}>{t(`fields.${f}`)}</option>)}
          </NativeSelect>
          <NativeSelect aria-label={t("direction")} value={action} onChange={(e) => setAction(e.target.value)} className="h-9">
            <option value="">{t("allDirections")}</option>
            {["to_ris", "to_portal", "review"].map((a) => <option key={a} value={a}>{t(`actions.${a}`)}</option>)}
          </NativeSelect>
        </div>
      </CardHeader>
      <CardContent className="overflow-x-auto">
        {loading ? (
          <p className="py-6 text-center text-sm text-muted-foreground">{t("loading")}</p>
        ) : items.length === 0 ? (
          <p className="py-6 text-center text-sm text-muted-foreground">{t("noChanges")}</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("researcher")}</TableHead>
                <TableHead>{t("field")}</TableHead>
                <TableHead>{t("direction")}</TableHead>
                <TableHead>{t("before")}</TableHead>
                <TableHead>{t("after")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((c, i) => (
                <TableRow key={i} className="align-top">
                  <TableCell className="whitespace-normal">
                    {c.researcher_id ? (
                      <Link href={`/researchers/${c.researcher_id}`} className="hover:underline"><Bidi text={c.researcher_name} /></Link>
                    ) : "—"}
                  </TableCell>
                  <TableCell>{t(`fields.${c.field}`)}</TableCell>
                  <TableCell>
                    <Badge variant="outline">{t(`actions.${c.action}`)}</Badge>
                    {c.note && <div className="mt-1 text-xs text-muted-foreground">{t.has(`notes.${c.note}`) ? t(`notes.${c.note}`) : c.note}</div>}
                  </TableCell>
                  <TableCell className="max-w-64 whitespace-normal"><ValueCell value={c.old_value} departments={departments} /></TableCell>
                  <TableCell className="max-w-64 whitespace-normal"><ValueCell value={c.new_value} departments={departments} /></TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}
