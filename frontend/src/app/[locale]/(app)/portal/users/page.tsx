"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { api } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { NativeSelect } from "@/components/portal/form-controls";
import { ResponsibilitiesDialog } from "@/components/portal/responsibilities-dialog";
import { RequireAuth } from "@/components/portal/require-auth";
import { usePortalErrors } from "@/components/portal/use-portal-errors";
import type { Org } from "@/lib/portal/types";
import { Alert } from "@/components/ui/alert";

interface AdminUser {
  id: string;
  email: string;
  name_ar: string | null;
  role: string;
  college_id: string | null;
  department_id: string | null;
  responsibilities_count: number;
}

const ROLES = [
  "admin", "vp_research", "scientific_affairs", "college_dean",
  "department_head", "researcher",
] as const;

function Users() {
  const t = useTranslations("portal");
  const tr = useTranslations("portal.responsibilities");
  const toMessage = usePortalErrors();
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [colleges, setColleges] = useState<Org[]>([]);
  const [departments, setDepartments] = useState<Org[]>([]);
  const [q, setQ] = useState("");
  const [savedId, setSavedId] = useState<string | null>(null);
  const [error, setError] = useState("");
  const [managing, setManaging] = useState<AdminUser | null>(null);

  useEffect(() => {
    let active = true;
    Promise.all([api.get<Org[]>("/api/v1/colleges"), api.get<Org[]>("/api/v1/departments")])
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
    let active = true;
    const timer = setTimeout(() => {
      api
        .get<AdminUser[]>(`/api/v1/admin/users?limit=50&q=${encodeURIComponent(q.trim())}`)
        .then((res) => active && setUsers(res.data ?? []))
        .catch((err) => active && setError(toMessage(err)));
    }, 250);
    return () => {
      active = false;
      clearTimeout(timer);
    };
  }, [q, toMessage]);

  function patch(id: string, change: Partial<AdminUser>) {
    setUsers((prev) => prev.map((u) => (u.id === id ? { ...u, ...change } : u)));
  }

  async function save(user: AdminUser) {
    setError("");
    setSavedId(null);
    try {
      await api.put(`/api/v1/admin/users/${user.id}`, {
        role: user.role,
        college_id: user.college_id ?? "",
        department_id: user.department_id ?? "",
      });
      setSavedId(user.id);
    } catch (err) {
      setError(toMessage(err));
    }
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t("users.title")}</h1>
      {error && <Alert variant="danger">{error}</Alert>}
      <Card>
        <CardContent className="space-y-4 pt-4">
          <Input placeholder={t("users.search")} value={q} onChange={(e) => setQ(e.target.value)} />
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("fields.name")}</TableHead>
                <TableHead>{t("fields.role")}</TableHead>
                <TableHead>{t("fields.college")}</TableHead>
                <TableHead>{t("fields.department")}</TableHead>
                <TableHead>{tr("manage")}</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {users.map((u) => (
                <TableRow key={u.id}>
                  <TableCell>
                    <div className="font-medium">{u.name_ar ?? "—"}</div>
                    <div className="text-xs text-muted-foreground" dir="ltr">{u.email}</div>
                  </TableCell>
                  <TableCell>
                    <NativeSelect value={u.role} onChange={(e) => patch(u.id, { role: e.target.value })}>
                      {ROLES.map((role) => (
                        <option key={role} value={role}>
                          {t(`users.roleNames.${role}`)}
                        </option>
                      ))}
                    </NativeSelect>
                  </TableCell>
                  <TableCell>
                    <NativeSelect
                      value={u.college_id ?? ""}
                      onChange={(e) => patch(u.id, { college_id: e.target.value || null, department_id: null })}
                    >
                      <option value="">{t("users.noOrg")}</option>
                      {colleges.map((c) => (
                        <option key={c.id} value={c.id}>{c.name_ar}</option>
                      ))}
                    </NativeSelect>
                  </TableCell>
                  <TableCell>
                    <NativeSelect
                      value={u.department_id ?? ""}
                      onChange={(e) => patch(u.id, { department_id: e.target.value || null })}
                    >
                      <option value="">{t("users.noOrg")}</option>
                      {departments
                        .filter((d) => !u.college_id || d.college_id === u.college_id)
                        .map((d) => (
                          <option key={d.id} value={d.id}>{d.name_ar}</option>
                        ))}
                    </NativeSelect>
                  </TableCell>
                  <TableCell className="whitespace-nowrap">
                    <Button size="sm" variant="outline" onClick={() => setManaging(u)}>
                      {tr("manage")} ({tr("count", { count: u.responsibilities_count })})
                    </Button>
                  </TableCell>
                  <TableCell className="whitespace-nowrap">
                    <Button size="sm" onClick={() => save(u)}>{t("users.save")}</Button>
                    {savedId === u.id && <span className="ms-2 text-xs text-status-success-fg">{t("users.saved")}</span>}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
      {managing && (
        <ResponsibilitiesDialog
          userId={managing.id}
          userName={managing.name_ar ?? managing.email}
          colleges={colleges}
          onClose={() => setManaging(null)}
          onSaved={(count) => patch(managing.id, { responsibilities_count: count })}
        />
      )}
    </div>
  );
}

export default function UsersPage() {
  return (
    <RequireAuth>
      <Users />
    </RequireAuth>
  );
}
