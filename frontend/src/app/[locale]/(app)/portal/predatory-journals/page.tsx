"use client";

import { useTranslations } from "next-intl";
import { ShieldAlert } from "lucide-react";
import { Alert } from "@/components/ui/alert";
import { PredatoryExplainer } from "@/components/journals/predatory-explainer";
import { ReferenceLinks } from "@/components/journals/reference-links";
import { RequireAuth } from "@/components/portal/require-auth";

/** Predatory journals: a short explanation and links to the trusted lists. */
function PredatoryJournalsPage() {
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
      <PredatoryExplainer />
      <h2 className="text-xl font-bold">{t("linksTitle")}</h2>
      <ReferenceLinks />
      <Alert variant="warning">{t("disclaimer")}</Alert>
    </div>
  );
}

export default function Page() {
  return (
    <RequireAuth>
      <PredatoryJournalsPage />
    </RequireAuth>
  );
}
