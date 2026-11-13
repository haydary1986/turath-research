"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { Plus, Trash2 } from "lucide-react";
import { api } from "@/lib/api";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { NativeSelect } from "@/components/portal/form-controls";
import { usePortalErrors } from "@/components/portal/use-portal-errors";
import { ACTIVITY_CATEGORIES, type Responsibility } from "@/lib/portal/activity";
import type { Org } from "@/lib/portal/types";

interface Row {
  key: number;
  category: string;
  college_id: string; // "" = whole university
}

interface ResponsibilitiesDialogProps {
  userId: string;
  userName: string;
  colleges: Org[];
  onClose: () => void;
  onSaved: (count: number) => void;
}

/** Admin control: which FS 003 categories a user enters, each for the university or one college. */
export function ResponsibilitiesDialog({ userId, userName, colleges, onClose, onSaved }: ResponsibilitiesDialogProps) {
  const tr = useTranslations("portal.responsibilities");
  const ta = useTranslations("portal.activities");
  const toMessage = usePortalErrors();
  const [rows, setRows] = useState<Row[]>([]);
  const [nextKey, setNextKey] = useState(0);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    let active = true;
    api
      .get<Responsibility[]>(`/api/v1/admin/users/${userId}/responsibilities`)
      .then((res) => {
        if (!active) return;
        const items = res.data ?? [];
        setRows(items.map((r, i) => ({ key: i, category: r.category, college_id: r.college_id ?? "" })));
        setNextKey(items.length);
      })
      .catch((err) => active && setError(toMessage(err)))
      .finally(() => active && setLoading(false));
    return () => {
      active = false;
    };
  }, [userId, toMessage]);

  function update(key: number, change: Partial<Row>) {
    setSaved(false);
    setRows((prev) => prev.map((r) => (r.key === key ? { ...r, ...change } : r)));
  }

  function add() {
    setSaved(false);
    setRows((prev) => [...prev, { key: nextKey, category: ACTIVITY_CATEGORIES[0], college_id: "" }]);
    setNextKey((k) => k + 1);
  }

  async function save() {
    setError("");
    setBusy(true);
    try {
      const res = await api.put<Responsibility[]>(`/api/v1/admin/users/${userId}/responsibilities`, {
        items: rows.map(({ category, college_id }) => ({ category, college_id })),
      });
      const items = res.data ?? [];
      setRows(items.map((r, i) => ({ key: i, category: r.category, college_id: r.college_id ?? "" })));
      setNextKey(items.length);
      setSaved(true);
      onSaved(items.length);
    } catch (err) {
      setError(toMessage(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog open onOpenChange={(open) => { if (!open && !busy) onClose(); }}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle className="pe-8">{tr("manageTitle", { name: userName })}</DialogTitle>
          <DialogDescription>{tr("manageHint")}</DialogDescription>
        </DialogHeader>
        {error && <Alert variant="danger" className="whitespace-pre-line">{error}</Alert>}
        {loading ? null : (
          <div className="space-y-3">
            {rows.length === 0 && <p className="text-sm text-muted-foreground">{tr("none")}</p>}
            {rows.map((row) => (
              <div key={row.key} className="grid gap-2 sm:grid-cols-[1fr_1fr_auto]">
                <NativeSelect aria-label={tr("category")} value={row.category}
                  onChange={(e) => update(row.key, { category: e.target.value })}>
                  {ACTIVITY_CATEGORIES.map((c) => <option key={c} value={c}>{ta(`categories.${c}`)}</option>)}
                </NativeSelect>
                <NativeSelect aria-label={tr("scope")} value={row.college_id}
                  onChange={(e) => update(row.key, { college_id: e.target.value })}>
                  <option value="">{tr("university")}</option>
                  {colleges.map((c) => <option key={c.id} value={c.id}>{c.name_ar}</option>)}
                </NativeSelect>
                <Button variant="outline" size="sm" aria-label={tr("remove")}
                  onClick={() => { setSaved(false); setRows((prev) => prev.filter((r) => r.key !== row.key)); }}>
                  <Trash2 className="size-4" />
                </Button>
              </div>
            ))}
            <div className="flex flex-wrap items-center gap-2">
              <Button variant="outline" size="sm" onClick={add}>
                <Plus className="size-4" />
                <span>{tr("add")}</span>
              </Button>
              <Button size="sm" disabled={busy} onClick={save}>{tr("save")}</Button>
              {saved && <span className="text-xs text-status-success-fg">{tr("saved")}</span>}
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
