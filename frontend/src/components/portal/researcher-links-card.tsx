"use client";

import { useEffect, useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import { ExternalLink } from "lucide-react";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useHydrated } from "@/hooks/use-hydrated";
import { needsIdentityReview, type ResearcherLinks } from "@/lib/portal/research-profile";
import { isStaffRole } from "@/lib/portal/types";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

const LINK_FIELDS = [
  ["orcid_url", "orcid"],
  ["google_scholar_url", "google_scholar_url"],
  ["scopus_url", "scopus_url"],
  ["researchgate_url", "researchgate_url"],
  ["linkedin_url", "linkedin_url"],
  ["web_of_science_url", "web_of_science_url"],
] as const;

function LinksCard({ researcherId }: { researcherId: string }) {
  const t = useTranslations("portal.profile");
  const fields = useTranslations("portal.fields");
  const locale = useLocale();
  const [links, setLinks] = useState<ResearcherLinks | null>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    let active = true;
    api
      .get<ResearcherLinks>(`/api/v1/researchers/${researcherId}/links`)
      .then((res) => active && setLinks(res.data ?? null))
      .catch(() => active && setFailed(true));
    return () => {
      active = false;
    };
  }, [researcherId]);

  if (failed) return <Alert variant="danger">{t("staffLinksError")}</Alert>;
  if (!links) return null;

  const present = LINK_FIELDS.filter(([key]) => links[key]);
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">{t("staffLinksTitle")}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {needsIdentityReview(links.last_sync_error) && (
          <Alert variant="warning">
            {t("staffIdentityReview", { reason: t(`syncErrors.${links.last_sync_error}`) })}
          </Alert>
        )}
        {present.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t("staffLinksEmpty")}</p>
        ) : (
          <div className="flex flex-wrap gap-2">
            {present.map(([key, label]) => (
              <Button
                key={key}
                variant="outline"
                size="sm"
                render={<a href={links[key] ?? undefined} target="_blank" rel="noopener noreferrer" />}
              >
                <span>{fields(label)}</span>
                <ExternalLink className="size-3.5" aria-hidden />
              </Button>
            ))}
          </div>
        )}
        <p className="text-xs text-muted-foreground">
          {links.last_synced_at
            ? t("lastSync", { time: new Date(links.last_synced_at).toLocaleString(locale) })
            : t("neverSynced")}
        </p>
      </CardContent>
    </Card>
  );
}

/** Links a researcher entered, for staff only; other viewers see nothing. */
export function ResearcherLinksCard({ researcherId }: { researcherId: string }) {
  const role = useAuth((state) => state.user?.role);
  const hydrated = useHydrated();
  if (!hydrated || !isStaffRole(role)) return null;
  return <LinksCard researcherId={researcherId} />;
}
