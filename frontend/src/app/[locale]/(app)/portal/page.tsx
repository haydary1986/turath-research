"use client";

import { useEffect, useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import {
  ArrowLeft, CheckCircle2, ClipboardList, Clock3, FileCheck2, FileText, Inbox, Mail,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { Link } from "@/i18n/routing";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { OrcidPrompt } from "@/components/portal/orcid-prompt";
import { RequireAuth } from "@/components/portal/require-auth";
import { RequestsTable } from "@/components/portal/requests-table";
import {
  REQUEST_KINDS,
  isStaffRole,
  type PortalRequest,
  type RequestKind,
  type StatCount,
} from "@/lib/portal/types";

const KIND_ICON: Record<RequestKind, LucideIcon> = {
  research_plan: FileText,
  journal_verification: FileCheck2,
  appreciation_letter: Mail,
};

const KIND_TINT: Record<RequestKind, string> = {
  research_plan: "bg-primary/10 text-primary",
  journal_verification: "bg-accent/10 text-accent",
  appreciation_letter: "bg-chart-4/10 text-chart-4",
};

const OPEN_STATUSES = ["pending_dean", "pending", "pending_vp", "awaiting_signatures"];
const DONE_STATUSES = ["approved", "verified", "completed", "issued"];

function Dashboard() {
  const t = useTranslations("portal");
  const locale = useLocale();
  const { user } = useAuth();
  const staff = isStaffRole(user?.role);
  const isDean = user?.role === "college_dean";

  const [stats, setStats] = useState<StatCount[]>([]);
  const [recent, setRecent] = useState<PortalRequest[]>([]);
  const [inbox, setInbox] = useState<PortalRequest[]>([]);
  const [inboxTotal, setInboxTotal] = useState(0);

  useEffect(() => {
    let active = true;
    const jobs: Promise<unknown>[] = [
      api.get<StatCount[]>("/api/v1/portal/stats").then((r) => active && setStats(r.data ?? [])),
      api
        .get<PortalRequest[]>("/api/v1/portal/requests?mine=true&limit=8")
        .then((r) => active && setRecent(r.data ?? [])),
    ];
    if (staff || isDean) {
      jobs.push(
        api
          .get<PortalRequest[]>("/api/v1/portal/requests?queue=true&limit=8")
          .then((r) => {
            if (!active) return;
            setInbox(r.data ?? []);
            setInboxTotal(r.meta?.total ?? 0);
          })
      );
    }
    Promise.all(jobs).catch(() => undefined);
    return () => {
      active = false;
    };
  }, [staff, isDean]);

  const count = (match: (s: StatCount) => boolean) =>
    stats.filter(match).reduce((sum, s) => sum + s.count, 0);
  const kindCount = (kind: RequestKind, statuses?: string[]) =>
    count((s) => s.kind === kind && (!statuses || statuses.includes(s.status)));

  const reviewer = staff || isDean;
  const summary = [
    ...(reviewer ? [{ label: t("ui.inbox"), value: inboxTotal, icon: Inbox }] : []),
    { label: t("ui.total"), value: count(() => true), icon: ClipboardList },
    { label: t("ui.openRequests"), value: count((s) => OPEN_STATUSES.includes(s.status)), icon: Clock3 },
    { label: t("ui.doneRequests"), value: count((s) => DONE_STATUSES.includes(s.status)), icon: CheckCircle2 },
  ].slice(0, 3);

  return (
    <div className="space-y-6">
      {/* Identity band */}
      <section className="overflow-hidden rounded-2xl border border-sidebar-border bg-gradient-to-l from-sidebar to-[oklch(0.239_0.101_263.3)] text-sidebar-foreground dark:to-[oklch(0.2_0.07_260)]">
        <div className="flex flex-wrap items-center justify-between gap-6 p-6 md:p-8">
          <div className="min-w-0">
            <p className="text-sm font-medium text-sidebar-foreground/85">{t("university")}</p>
            <h1 className="mt-1 text-2xl font-bold md:text-3xl">{t("title")}</h1>
            <p className="mt-2 max-w-xl text-sm leading-relaxed opacity-80">{t("subtitle")}</p>
          </div>
          <div className="flex flex-wrap gap-3">
            {summary.map(({ label, value, icon: Icon }) => (
              <div
                key={label}
                className="min-w-24 rounded-xl bg-sidebar-accent/60 px-4 py-3 text-center"
              >
                <Icon className="mx-auto size-4 text-sidebar-primary" aria-hidden />
                <p className="mt-1 text-2xl font-bold tabular-nums">{value}</p>
                <p className="text-xs opacity-80">{label}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {!staff && <OrcidPrompt />}

      {reviewer && (
        <Card className="border-accent/30">
          <CardHeader className="flex-row items-center justify-between">
            <CardTitle className="flex items-center gap-2">
              {t("ui.inbox")}
              {inboxTotal > 0 && <Badge className="tabular-nums">{inboxTotal}</Badge>}
            </CardTitle>
            {inboxTotal > 0 && (
              <Button variant="outline" size="sm"
                render={<Link href="/portal/requests?queue=true" />}>
                {t("ui.viewAll")}
              </Button>
            )}
          </CardHeader>
          <CardContent>
            {inbox.length === 0 ? (
              <p className="py-6 text-center text-sm text-muted-foreground">{t("ui.inboxEmpty")}</p>
            ) : (
              <RequestsTable items={inbox} locale={locale} showApplicant />
            )}
          </CardContent>
        </Card>
      )}

      {/* Service cards */}
      <div className="grid gap-4 md:grid-cols-3">
        {REQUEST_KINDS.map((kind) => {
          const Icon = KIND_ICON[kind];
          return (
            <Card key={kind} className="flex flex-col transition-shadow hover:shadow-md">
              <CardHeader>
                <div className={`flex size-11 items-center justify-center rounded-xl ${KIND_TINT[kind]}`}>
                  <Icon className="size-5" aria-hidden />
                </div>
                <CardTitle className="mt-3 text-base">{t(`kinds.${kind}`)}</CardTitle>
              </CardHeader>
              <CardContent className="flex flex-1 flex-col gap-4">
                <p className="text-sm leading-relaxed text-muted-foreground">{t(`kindDesc.${kind}`)}</p>
                <div className="mt-auto flex items-center justify-between border-t pt-3 text-sm">
                  <span className="text-muted-foreground">
                    {t("ui.total")} <b className="tabular-nums text-foreground">{kindCount(kind)}</b>
                  </span>
                  <span className="text-muted-foreground">
                    {t("ui.openShort")}{" "}
                    <b className="tabular-nums text-foreground">{kindCount(kind, OPEN_STATUSES)}</b>
                  </span>
                </div>
                <Button className="w-full" render={<Link href={`/portal/new/${kind}`} />}>
                  <span>{t("ui.newRequest")}</span>
                  <ArrowLeft className="size-4 ltr:rotate-180" aria-hidden />
                </Button>
              </CardContent>
            </Card>
          );
        })}
      </div>

      <Card>
        <CardHeader className="flex-row items-center justify-between">
          <CardTitle>{t("ui.myRequests")}</CardTitle>
          <Button variant="outline" size="sm" render={<Link href="/portal/requests" />}>
            {t("ui.allRequests")}
          </Button>
        </CardHeader>
        <CardContent>
          <RequestsTable items={recent} locale={locale} />
        </CardContent>
      </Card>
    </div>
  );
}

export default function PortalHomePage() {
  return (
    <RequireAuth>
      <Dashboard />
    </RequireAuth>
  );
}
