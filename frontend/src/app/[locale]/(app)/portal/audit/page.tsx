"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { AccountsActivity } from "@/components/portal/accounts-activity";
import { LoginLog } from "@/components/portal/login-log";
import { RequireAuth } from "@/components/portal/require-auth";
import { useAuth } from "@/lib/auth";

const TABS = ["log", "accounts"] as const;

/** «سجل الدخول والحسابات» (admin only; the API refuses everyone else). */
function AuditPage() {
  const tu = useTranslations("portal.audit");
  const errors = useTranslations("portal.errors");
  const role = useAuth((s) => s.user?.role);
  const [tab, setTab] = useState<(typeof TABS)[number]>("log");

  if (role !== "admin") return <Alert variant="danger">{errors("generic")}</Alert>;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{tu("title")}</h1>
        <p className="text-sm text-muted-foreground">{tu("intro")}</p>
      </div>
      <div className="flex gap-2" role="tablist">
        {TABS.map((key) => (
          <Button key={key} role="tab" aria-selected={tab === key} variant={tab === key ? "default" : "outline"}
            onClick={() => setTab(key)}>
            {tu(`tabs.${key}`)}
          </Button>
        ))}
      </div>
      <Card>
        <CardContent className="pt-4">{tab === "log" ? <LoginLog /> : <AccountsActivity />}</CardContent>
      </Card>
    </div>
  );
}

export default function Page() {
  return (
    <RequireAuth>
      <AuditPage />
    </RequireAuth>
  );
}
