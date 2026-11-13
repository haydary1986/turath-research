"use client";

import { useTranslations } from "next-intl";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { NativeSelect } from "@/components/portal/form-controls";
import { SocialIcon } from "@/components/site/social-icon";
import { SITE_LIMITS, SOCIAL_PLATFORMS, type SiteSocial, type SocialPlatform } from "@/lib/site/types";
import { Field, ListHeader, RowControls, move, removeAt, replaceAt } from "./editor-parts";

interface SocialsEditorProps {
  socials: SiteSocial[];
  onChange: (socials: SiteSocial[]) => void;
}

export function SocialsEditor({ socials, onChange }: SocialsEditorProps) {
  const t = useTranslations("siteEditor.socials");
  const tn = useTranslations("landing.footer.social");
  const tc = useTranslations("portal.site.content");
  const set = (i: number, patch: Partial<SiteSocial>) => onChange(replaceAt(socials, i, { ...socials[i], ...patch }));

  return (
    <Card>
      <ListHeader title={t("title")} hint={t("hint")} addLabel={t("add")} canAdd={socials.length < SITE_LIMITS.socials}
        onAdd={() => onChange([...socials, { platform: "facebook", url: "https://" }])} />
      <CardContent className="space-y-3">
        {socials.map((social, i) => (
          <fieldset key={i} className="grid items-end gap-2 rounded-lg border p-3 sm:grid-cols-[auto_12rem_1fr_auto]">
            <legend className="sr-only">{tc("itemN", { n: i + 1 })}</legend>
            <span className="mb-1 flex size-8 items-center justify-center rounded-full bg-primary text-primary-foreground" aria-hidden>
              <SocialIcon platform={social.platform} className="size-4" />
            </span>
            <Field id={`social_platform_${i}`} label={t("platform")}>
              <NativeSelect id={`social_platform_${i}`} value={social.platform}
                onChange={(e) => set(i, { platform: e.target.value as SocialPlatform })}>
                {SOCIAL_PLATFORMS.map((p) => <option key={p} value={p}>{tn(p)}</option>)}
              </NativeSelect>
            </Field>
            <Field id={`social_url_${i}`} label={t("url")}>
              <Input id={`social_url_${i}`} type="url" dir="ltr" required pattern="https://.+" maxLength={SITE_LIMITS.linkUrl}
                value={social.url} onChange={(e) => set(i, { url: e.target.value })} />
            </Field>
            <RowControls index={i} count={socials.length}
              onMove={(to) => onChange(move(socials, i, to))} onRemove={() => onChange(removeAt(socials, i))} />
          </fieldset>
        ))}
      </CardContent>
    </Card>
  );
}
