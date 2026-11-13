"use client";

import { useTranslations } from "next-intl";
import { ExternalLink, SearchCheck, ShieldAlert } from "lucide-react";
import { Link } from "@/i18n/routing";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { PredatoryListBrowser } from "@/components/portal/predatory-list-browser";
import { RequireAuth } from "@/components/portal/require-auth";

const SIGNS = ["sign1", "sign2", "sign3", "sign4", "sign5", "sign6", "sign7"] as const;
const STEPS = ["step1", "step2", "step3", "step4"] as const;

const SOURCE_LINKS = [
  { key: "home", href: "https://beallslist.net/" },
  { key: "journals", href: "https://beallslist.net/standalone-journals/" },
  { key: "hijacked", href: "https://beallslist.net/hijacked-journals/" },
  { key: "recognize", href: "https://beallslist.net/how-to-recognize-predatory-journals/" },
  { key: "metrics", href: "https://beallslist.net/misleading-metrics/" },
  { key: "pjHome", href: "https://www.predatoryjournals.org/home" },
  { key: "pjJournals", href: "https://www.predatoryjournals.org/the-list/journals" },
  { key: "pjPublishers", href: "https://www.predatoryjournals.org/the-list/publishers" },
] as const;

function PredatoryGuidePage() {
  const t = useTranslations("portal.predatory");

  return (
    <div className="space-y-6">
      <div className="flex items-start gap-3">
        <div className="flex size-11 shrink-0 items-center justify-center rounded-xl bg-status-danger-bg text-status-danger-fg">
          <ShieldAlert className="size-5" aria-hidden />
        </div>
        <div>
          <h1 className="text-2xl font-bold">{t("title")}</h1>
          <p className="text-sm text-muted-foreground">{t("subtitle")}</p>
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>{t("whatTitle")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-sm leading-relaxed">
            <p>{t("what")}</p>
            <p className="text-muted-foreground">{t("whyMatters")}</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{t("stepsTitle")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            <ol className="list-decimal space-y-2 ps-5 leading-relaxed">
              {STEPS.map((s) => (
                <li key={s}>{t(s)}</li>
              ))}
            </ol>
            <Button render={<Link href="/portal/journal-check" />}>
              <SearchCheck className="size-4" aria-hidden />
              {t("checkButton")}
            </Button>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t("signsTitle")}</CardTitle>
        </CardHeader>
        <CardContent>
          <ul className="grid gap-2 text-sm leading-relaxed md:grid-cols-2">
            {SIGNS.map((s) => (
              <li key={s} className="flex gap-2 rounded-lg bg-muted p-3">
                <span className="mt-1.5 size-1.5 shrink-0 rounded-full bg-status-danger" aria-hidden />
                <span>{t(s)}</span>
              </li>
            ))}
          </ul>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("listsTitle")}</CardTitle>
          <p className="text-sm text-muted-foreground">{t("listsIntro")}</p>
        </CardHeader>
        <CardContent>
          <PredatoryListBrowser />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("sourcesTitle")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <ul className="flex flex-wrap gap-2">
            {SOURCE_LINKS.map((l) => (
              <li key={l.key}>
                <Button
                  variant="outline"
                  size="sm"
                  render={<a href={l.href} target="_blank" rel="noopener noreferrer" />}
                >
                  <ExternalLink className="size-4" aria-hidden />
                  {t(`links.${l.key}`)}
                </Button>
              </li>
            ))}
          </ul>
          <Alert variant="warning">{t("disclaimer")}</Alert>
        </CardContent>
      </Card>
    </div>
  );
}

export default function Page() {
  return (
    <RequireAuth>
      <PredatoryGuidePage />
    </RequireAuth>
  );
}
