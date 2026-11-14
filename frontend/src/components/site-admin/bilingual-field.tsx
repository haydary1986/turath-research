"use client";

import { useTranslations } from "next-intl";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/portal/form-controls";
import { cn } from "@/lib/utils";
import { CharCount } from "./char-count";

interface BilingualFieldProps {
  /** Base id; the inputs get `${id}` (Arabic) and `${id}_en` (English). */
  id: string;
  label: string;
  ar: string;
  en: string;
  onChange: (patch: { ar?: string; en?: string }) => void;
  max: number;
  multiline?: boolean;
  rows?: number;
  /** The Arabic value is required; English is always optional. */
  required?: boolean;
  hint?: string;
  /** Placeholders, e.g. the default label a statistic falls back to. */
  placeholderAr?: string;
  placeholderEn?: string;
  /** Show the "used / max" counter under each control. */
  counter?: boolean;
  className?: string;
}

/**
 * One editable text in both languages, side by side: Arabic (primary, laid out
 * right-to-left) and English (optional, left-to-right). The English site shows the
 * Arabic text wherever the English one is left empty.
 */
export function BilingualField({
  id, label, ar, en, onChange, max, multiline = false, rows = 3, required = false, hint,
  placeholderAr, placeholderEn, counter = multiline, className,
}: BilingualFieldProps) {
  const t = useTranslations("siteEditor.bilingual");
  const sides = [
    { lang: "ar" as const, dir: "rtl" as const, inputId: id, value: ar, label: t("arabic", { label }), placeholder: placeholderAr, required },
    { lang: "en" as const, dir: "ltr" as const, inputId: `${id}_en`, value: en, label: t("english", { label }), placeholder: placeholderEn, required: false },
  ];

  return (
    <div className={cn("space-y-1.5", className)}>
      <div className="grid gap-3 md:grid-cols-2">
        {sides.map((side) => (
          <div key={side.lang} className="space-y-1.5">
            <Label htmlFor={side.inputId}>
              {side.label}
              {side.lang === "en" && <span className="text-xs font-normal text-muted-foreground">{t("optional")}</span>}
            </Label>
            {multiline ? (
              <Textarea id={side.inputId} lang={side.lang} dir={side.dir} rows={rows} required={side.required}
                value={side.value} placeholder={side.placeholder}
                onChange={(e) => onChange({ [side.lang]: e.target.value })} />
            ) : (
              <Input id={side.inputId} lang={side.lang} dir={side.dir} required={side.required} maxLength={max}
                value={side.value} placeholder={side.placeholder}
                onChange={(e) => onChange({ [side.lang]: e.target.value })} />
            )}
            {counter && <CharCount value={side.value} max={max} />}
          </div>
        ))}
      </div>
      {hint && <p className="text-xs text-muted-foreground">{hint}</p>}
    </div>
  );
}
