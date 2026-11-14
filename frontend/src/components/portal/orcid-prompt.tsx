"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { Fingerprint } from "lucide-react";
import { Link } from "@/i18n/routing";
import { api } from "@/lib/api";
import { PROFILE_ENDPOINT, type ResearchProfile } from "@/lib/portal/research-profile";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";

/** Dashboard nudge shown until the researcher has saved an ORCID iD. */
export function OrcidPrompt() {
  const t = useTranslations("portal.profile");
  const [missing, setMissing] = useState(false);

  useEffect(() => {
    let active = true;
    api
      .get<ResearchProfile>(PROFILE_ENDPOINT)
      .then((res) => active && setMissing(!res.data?.orcid))
      .catch(() => undefined);
    return () => {
      active = false;
    };
  }, []);

  if (!missing) return null;
  return (
    <Card className="border-primary/30">
      <CardContent className="flex flex-wrap items-center justify-between gap-4">
        <div className="flex min-w-0 items-start gap-3">
          <Fingerprint className="mt-0.5 size-5 shrink-0 text-primary" aria-hidden />
          <div className="min-w-0">
            <p className="font-medium">{t("promptTitle")}</p>
            <p className="text-sm text-muted-foreground">{t("promptBody")}</p>
          </div>
        </div>
        <Button nativeButton={false} render={<Link href="/portal/profile" />}>{t("promptAction")}</Button>
      </CardContent>
    </Card>
  );
}
