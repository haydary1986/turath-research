"use client";

import { useTranslations } from "next-intl";
import { JournalCheckSearch } from "@/components/portal/journal-check-search";
import { RequireAuth } from "@/components/portal/require-auth";

function JournalCheckPage() {
  const t = useTranslations("portal.journals.check");
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{t("title")}</h1>
        <p className="text-sm text-muted-foreground">{t("pageSubtitle")}</p>
      </div>
      <JournalCheckSearch />
    </div>
  );
}

export default function Page() {
  return (
    <RequireAuth>
      <JournalCheckPage />
    </RequireAuth>
  );
}
