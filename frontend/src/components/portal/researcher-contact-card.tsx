"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { api } from "@/lib/api";
import { useHydrated } from "@/hooks/use-hydrated";
import { contactEntries, type ContactEntry, type ResearcherContact } from "@/lib/portal/research-profile";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

const HREF_PREFIX = { tel: "tel:", mailto: "mailto:" } as const;

function ContactValue({ entry }: { entry: ContactEntry }) {
  const value = <bdi dir="ltr">{entry.value}</bdi>;
  if (entry.kind === "text") return value;
  return (
    <a className="text-primary hover:underline" href={`${HREF_PREFIX[entry.kind]}${entry.value}`}>
      {value}
    </a>
  );
}

/**
 * The server decides who may read contact data (the researcher, scientific affairs, the
 * VP, admin), and the client cannot tell whether a page is the viewer's own. So the card
 * asks once, and any refusal simply means it stays hidden.
 */
export function ResearcherContactCard({ researcherId }: { researcherId: string }) {
  const hydrated = useHydrated();
  const t = useTranslations("portal.profile");
  const fields = useTranslations("portal.fields");
  const [contact, setContact] = useState<ResearcherContact | null>(null);

  useEffect(() => {
    if (!hydrated) return;
    let active = true;
    api
      .get<ResearcherContact>(`/api/v1/researchers/${researcherId}/contact`)
      .then((res) => active && setContact(res.data ?? null))
      .catch(() => active && setContact(null));
    return () => {
      active = false;
    };
  }, [hydrated, researcherId]);

  if (!contact) return null;

  const label = (key: ContactEntry["key"]) => (fields.has(key) ? fields(key) : t(key));
  const statusKey = `accountStatus.${contact.account_status}`;
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">{t("contactTitle")}</CardTitle>
      </CardHeader>
      <CardContent>
        <dl className="grid gap-3 text-sm sm:grid-cols-2 lg:grid-cols-3">
          {contactEntries(contact).map((entry) => (
            <div key={entry.key} className="min-w-0">
              <dt className="text-xs text-muted-foreground">{label(entry.key)}</dt>
              <dd className="break-words">
                <ContactValue entry={entry} />
              </dd>
            </div>
          ))}
          {contact.account_status && t.has(statusKey) && (
            <div>
              <dt className="text-xs text-muted-foreground">{t("contactAccountStatus")}</dt>
              <dd>{t(statusKey)}</dd>
            </div>
          )}
        </dl>
      </CardContent>
    </Card>
  );
}
