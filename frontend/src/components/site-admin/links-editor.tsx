"use client";

import { useTranslations } from "next-intl";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { SITE_LIMITS, type SiteLink } from "@/lib/site/types";
import { Field, ListHeader, RowControls, move, removeAt, replaceAt } from "./editor-parts";

interface LinksEditorProps {
  /** Prefix for input ids, unique per editor on the page. */
  idPrefix: string;
  title: string;
  hint?: string;
  addLabel: string;
  labelLabel: string;
  urlLabel: string;
  max: number;
  links: SiteLink[];
  onChange: (links: SiteLink[]) => void;
}

/** A list of labelled https links (quick links, college sites). */
export function LinksEditor({ idPrefix, title, hint, addLabel, labelLabel, urlLabel, max, links, onChange }: LinksEditorProps) {
  const tc = useTranslations("portal.site.content");
  const set = (i: number, patch: Partial<SiteLink>) => onChange(replaceAt(links, i, { ...links[i], ...patch }));

  return (
    <Card>
      <ListHeader title={title} hint={hint} addLabel={addLabel} canAdd={links.length < max}
        onAdd={() => onChange([...links, { label: "", url: "https://" }])} />
      <CardContent className="space-y-3">
        {links.map((link, i) => (
          <fieldset key={i} className="grid items-end gap-2 rounded-lg border p-3 sm:grid-cols-[1fr_1.4fr_auto]">
            <legend className="sr-only">{tc("itemN", { n: i + 1 })}</legend>
            <Field id={`${idPrefix}_label_${i}`} label={labelLabel}>
              <Input id={`${idPrefix}_label_${i}`} required maxLength={SITE_LIMITS.linkLabel} value={link.label}
                onChange={(e) => set(i, { label: e.target.value })} />
            </Field>
            <Field id={`${idPrefix}_url_${i}`} label={urlLabel}>
              <Input id={`${idPrefix}_url_${i}`} type="url" dir="ltr" required pattern="https://.+" maxLength={SITE_LIMITS.linkUrl}
                value={link.url} onChange={(e) => set(i, { url: e.target.value })} />
            </Field>
            <RowControls index={i} count={links.length}
              onMove={(to) => onChange(move(links, i, to))} onRemove={() => onChange(removeAt(links, i))} />
          </fieldset>
        ))}
      </CardContent>
    </Card>
  );
}
