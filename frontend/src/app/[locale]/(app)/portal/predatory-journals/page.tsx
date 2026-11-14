"use client";

import { useTranslations } from "next-intl";
import { SearchCheck, ShieldAlert } from "lucide-react";
import { Link } from "@/i18n/routing";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { ReferenceLinks } from "@/components/journals/reference-links";
import { RequireAuth } from "@/components/portal/require-auth";

/** Links to the predatory, hijacked and ministry journal lists, plus the journal check. */
function PredatoryLinksPage() {
  const t = useTranslations("journals.refLinks");

  return (
    <div className="space-y-6">
      <div className="flex items-start gap-3">
        <div className="flex size-11 shrink-0 items-center justify-center rounded-xl bg-status-danger-bg text-status-danger-fg">
          <ShieldAlert className="size-5" aria-hidden />
        </div>
        <div>
          <h1 className="text-2xl font-bold">{t("title")}</h1>
          <p className="text-sm text-muted-foreground">{t("intro")}</p>
        </div>
      </div>

      <ReferenceLinks />

      <div className="flex flex-wrap items-center gap-3 rounded-xl border bg-muted/40 p-4">
        <p className="flex-1 text-sm text-muted-foreground">{t("checkHint")}</p>
        <Button nativeButton={false} render={<Link href="/portal/journal-check" />}>
          <SearchCheck className="size-4" aria-hidden />
          {t("checkButton")}
        </Button>
      </div>
      <Alert variant="warning">{t("disclaimer")}</Alert>
    </div>
  );
}

export default function Page() {
  return (
    <RequireAuth>
      <PredatoryLinksPage />
    </RequireAuth>
  );
}
