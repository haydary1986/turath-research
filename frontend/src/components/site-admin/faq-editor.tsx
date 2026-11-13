"use client";

import { useTranslations } from "next-intl";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/portal/form-controls";
import { SITE_LIMITS, type SiteFAQItem } from "@/lib/site/types";
import { CharCount } from "./char-count";
import { Field, ListHeader, RowControls, move, removeAt, replaceAt } from "./editor-parts";

interface FaqEditorProps {
  items: SiteFAQItem[];
  onChange: (items: SiteFAQItem[]) => void;
}

export function FaqEditor({ items, onChange }: FaqEditorProps) {
  const t = useTranslations("siteEditor.faq");
  const tc = useTranslations("portal.site.content");
  const tp = useTranslations("portal.site");
  const set = (i: number, patch: Partial<SiteFAQItem>) => onChange(replaceAt(items, i, { ...items[i], ...patch }));

  return (
    <Card>
      <ListHeader title={t("title")} hint={t("hint")} addLabel={t("add")} canAdd={items.length < SITE_LIMITS.faq}
        onAdd={() => onChange([...items, { question: "", answer: "" }])} />
      <CardContent className="space-y-3">
        {items.map((item, i) => (
          <fieldset key={i} className="space-y-3 rounded-lg border p-3">
            <legend className="sr-only">{tc("itemN", { n: i + 1 })}</legend>
            <div className="flex items-start gap-2">
              <div className="flex-1">
                <Field id={`faq_q_${i}`} label={t("question")}>
                  <Input id={`faq_q_${i}`} required maxLength={SITE_LIMITS.faqQuestion} value={item.question}
                    onChange={(e) => set(i, { question: e.target.value })} />
                </Field>
              </div>
              <RowControls index={i} count={items.length}
                onMove={(to) => onChange(move(items, i, to))} onRemove={() => onChange(removeAt(items, i))} />
            </div>
            <Field id={`faq_a_${i}`} label={t("answer")} hint={tp("formatHint")}>
              <Textarea id={`faq_a_${i}`} required rows={3} value={item.answer} onChange={(e) => set(i, { answer: e.target.value })} />
              <CharCount value={item.answer} max={SITE_LIMITS.faqAnswer} />
            </Field>
          </fieldset>
        ))}
      </CardContent>
    </Card>
  );
}
