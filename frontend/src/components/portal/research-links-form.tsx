"use client";

import type { FormEvent } from "react";
import { useTranslations } from "next-intl";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Ltr } from "@/components/i18n/bidi";

export interface LinksFormValues {
  orcid: string;
  google_scholar_url: string;
  scopus_url: string;
  researchgate_url: string;
  linkedin_url: string;
  web_of_science_url: string;
  phone: string;
  phone_secondary: string;
  recovery_email: string;
}

const LINK_ONLY_FIELDS: { key: keyof LinksFormValues; example: string }[] = [
  { key: "google_scholar_url", example: "https://scholar.google.com/citations?user=AbCdEfGhIjk" },
  { key: "scopus_url", example: "https://www.scopus.com/authid/detail.uri?authorId=57193000000" },
  { key: "researchgate_url", example: "https://www.researchgate.net/profile/Your-Name" },
  { key: "linkedin_url", example: "https://www.linkedin.com/in/your-name" },
  { key: "web_of_science_url", example: "https://www.webofscience.com/wos/author/record/1234567" },
];
const CONTACT_FIELDS: { key: keyof LinksFormValues; example: string; type: string; mode: "tel" | "email" }[] = [
  { key: "phone", example: "07XXXXXXXXX", type: "tel", mode: "tel" },
  { key: "phone_secondary", example: "07XXXXXXXXX", type: "tel", mode: "tel" },
  { key: "recovery_email", example: "name@example.com", type: "email", mode: "email" },
];
const ORCID_EXAMPLE = "0000-0002-1825-0097";

interface Props {
  values: LinksFormValues;
  saving: boolean;
  onChange: (key: keyof LinksFormValues, value: string) => void;
  onSubmit: () => void;
}

export function ResearchLinksForm({ values, saving, onChange, onSubmit }: Props) {
  const t = useTranslations("portal.profile");
  const fields = useTranslations("portal.fields");

  const submit = (event: FormEvent) => {
    event.preventDefault();
    onSubmit();
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("linksTitle")}</CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={submit} className="space-y-5">
          <div className="space-y-1.5 rounded-lg border border-primary/30 bg-primary/5 p-3">
            <div className="flex flex-wrap items-center gap-2">
              <Label htmlFor="orcid">{fields("orcid")}</Label>
              <Badge>{t("automaticBadge")}</Badge>
            </div>
            <Input
              id="orcid"
              dir="ltr"
              autoComplete="off"
              placeholder={ORCID_EXAMPLE}
              value={values.orcid}
              onChange={(e) => onChange("orcid", e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              {t("example")} <Ltr>{ORCID_EXAMPLE}</Ltr>
            </p>
            <p className="text-xs font-medium text-foreground">{t("hintOrcid")}</p>
          </div>

          <div className="grid gap-5 md:grid-cols-2">
            {LINK_ONLY_FIELDS.map(({ key, example }) => (
              <div key={key} className="min-w-0 space-y-1.5">
                <Label htmlFor={key}>{fields(key)}</Label>
                <Input
                  id={key}
                  dir="ltr"
                  autoComplete="off"
                  placeholder={example}
                  value={values[key]}
                  onChange={(e) => onChange(key, e.target.value)}
                />
                <p className="text-xs text-muted-foreground">{t("hintLinkOnly")}</p>
              </div>
            ))}
          </div>

          <fieldset className="space-y-3 rounded-lg border p-3">
            <legend className="px-1 text-sm font-medium">{t("contactFormTitle")}</legend>
            <p className="text-xs text-muted-foreground">{t("contactFormHint")}</p>
            <div className="grid gap-5 md:grid-cols-3">
              {CONTACT_FIELDS.map(({ key, example, type, mode }) => (
                <div key={key} className="min-w-0 space-y-1.5">
                  <Label htmlFor={key}>{fields(key)}</Label>
                  <Input
                    id={key}
                    dir="ltr"
                    type={type}
                    inputMode={mode}
                    autoComplete="off"
                    placeholder={example}
                    value={values[key]}
                    onChange={(e) => onChange(key, e.target.value)}
                  />
                </div>
              ))}
            </div>
          </fieldset>

          <Button type="submit" disabled={saving}>
            {saving ? t("saving") : t("save")}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
