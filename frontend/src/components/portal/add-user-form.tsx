"use client";

import { useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import { UserPlus } from "lucide-react";
import { api } from "@/lib/api";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { Org } from "@/lib/portal/types";
import { NativeSelect } from "./form-controls";
import { usePortalErrors } from "./use-portal-errors";
import { DataOption } from "@/components/i18n/data-option";

export interface CreatedUser {
  id: string;
  email: string;
  name_ar: string | null;
  role: string;
  college_id: string | null;
  department_id: string | null;
}

interface AddUserFormProps {
  roles: readonly string[];
  colleges: Org[];
  departments: Org[];
  onCreated: (user: CreatedUser) => void;
}

const EMPTY = { email: "", name_ar: "", role: "researcher", college_id: "", department_id: "" };

/** Pre-registers a university account: its owner signs in with Google and finds the role ready. */
export function AddUserForm({ roles, colleges, departments, onCreated }: AddUserFormProps) {
  const t = useTranslations("portal");
  const locale = useLocale();
  const toMessage = usePortalErrors();
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState(EMPTY);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setBusy(true);
    try {
      const res = await api.post<CreatedUser>("/api/v1/admin/users", form);
      if (res.data) onCreated(res.data);
      setForm(EMPTY);
      setOpen(false);
    } catch (err) {
      setError(toMessage(err));
    } finally {
      setBusy(false);
    }
  }

  if (!open) {
    return (
      <Button onClick={() => setOpen(true)}>
        <UserPlus className="size-4" aria-hidden />
        {t("users.add")}
      </Button>
    );
  }

  const visibleDepartments = departments.filter((d) => !form.college_id || d.college_id === form.college_id);

  return (
    <Card className="border-primary/30">
      <CardHeader>
        <CardTitle>{t("users.add")}</CardTitle>
        <p className="text-sm text-muted-foreground">{t("users.addHint")}</p>
      </CardHeader>
      <CardContent>
        <form onSubmit={submit} className="grid gap-4 md:grid-cols-2">
          {error && <Alert variant="danger" className="md:col-span-2">{error}</Alert>}
          <div className="space-y-2">
            <Label htmlFor="new-user-email">{t("fields.email")} *</Label>
            <Input
              id="new-user-email"
              type="email"
              dir="ltr"
              required
              placeholder="name@uoturath.edu.iq"
              value={form.email}
              onChange={(e) => setForm({ ...form, email: e.target.value })}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="new-user-name">{t("fields.name")} *</Label>
            <Input
              id="new-user-name"
              required
              value={form.name_ar}
              onChange={(e) => setForm({ ...form, name_ar: e.target.value })}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="new-user-role">{t("fields.role")} *</Label>
            <NativeSelect id="new-user-role" value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value })}>
              {roles.map((role) => (
                <option key={role} value={role}>
                  {t(`users.roleNames.${role}`)}
                </option>
              ))}
            </NativeSelect>
          </div>
          <div className="space-y-2">
            <Label htmlFor="new-user-college">{t("fields.college")}</Label>
            <NativeSelect
              id="new-user-college"
              value={form.college_id}
              onChange={(e) => setForm({ ...form, college_id: e.target.value, department_id: "" })}
            >
              <option value="">{t("users.noOrg")}</option>
              {colleges.map((c) => (
                <DataOption key={c.id} value={c.id} ar={c.name_ar} en={c.name_en} locale={locale} />
              ))}
            </NativeSelect>
          </div>
          <div className="space-y-2">
            <Label htmlFor="new-user-department">{t("fields.department")}</Label>
            <NativeSelect
              id="new-user-department"
              value={form.department_id}
              onChange={(e) => setForm({ ...form, department_id: e.target.value })}
            >
              <option value="">{t("users.noOrg")}</option>
              {visibleDepartments.map((d) => (
                <DataOption key={d.id} value={d.id} ar={d.name_ar} en={d.name_en} locale={locale} />
              ))}
            </NativeSelect>
          </div>
          <div className="flex items-end gap-2">
            <Button type="submit" disabled={busy}>
              {busy ? t("ui.saving") : t("users.create")}
            </Button>
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              {t("users.cancel")}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
