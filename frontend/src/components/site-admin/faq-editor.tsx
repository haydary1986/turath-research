"use client";

import { useTranslations } from "next-intl";
import { Card, CardContent } from "@/components/ui/card";
import { SITE_LIMITS, type SiteFAQItem } from "@/lib/site/types";
import { BilingualField } from "./bilingual-field";
import { ListHeader, RowControls, move, removeAt, replaceAt } from "./editor-parts";

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
        onAdd={() => onChange([...items, { question: "", question_en: "", answer: "", answer_en: "" }])} />
      <CardContent className="space-y-3">
        {items.map((item, i) => (
          <fieldset key={i} className="space-y-3 rounded-lg border p-3">
            <legend className="sr-only">{tc("itemN", { n: i + 1 })}</legend>
            <div className="flex items-start gap-2">
              <BilingualField id={`faq_q_${i}`} label={t("question")} required max={SITE_LIMITS.faqQuestion} className="flex-1"
                ar={item.question} en={item.question_en}
                onChange={(v) => set(i, { question: v.ar ?? item.question, question_en: v.en ?? item.question_en })} />
              <RowControls index={i} count={items.length}
                onMove={(to) => onChange(move(items, i, to))} onRemove={() => onChange(removeAt(items, i))} />
            </div>
            <BilingualField id={`faq_a_${i}`} label={t("answer")} required multiline rows={3} max={SITE_LIMITS.faqAnswer}
              hint={tp("formatHint")} ar={item.answer} en={item.answer_en}
              onChange={(v) => set(i, { answer: v.ar ?? item.answer, answer_en: v.en ?? item.answer_en })} />
          </fieldset>
        ))}
      </CardContent>
    </Card>
  );
}
