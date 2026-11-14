"use client";

import { useCallback } from "react";
import { useTranslations } from "next-intl";
import { parseValidation } from "@/lib/portal/types";

/** Localises API errors from the site endpoints, labelling fields with the site editor's names. */
export function useSiteErrors(): (error: unknown) => string {
  const errors = useTranslations("portal.errors");
  const fields = useTranslations("portal.site.fields");
  const blockFields = useTranslations("siteEditor.fields");
  const bilingual = useTranslations("siteEditor.bilingual");

  const fieldLabel = useCallback(
    (field: string): string => {
      const named = (key: string): string | null =>
        fields.has(key) ? fields(key) : blockFields.has(key) ? blockFields(key) : null;
      // English counterparts ("hero_title_en") are labelled as "<field> — English".
      if (field.endsWith("_en")) {
        const base = named(field.slice(0, -3));
        if (base) return bilingual("english", { label: base });
      }
      return named(field) ?? field;
    },
    [fields, blockFields, bilingual]
  );

  return useCallback(
    (error: unknown): string => {
      const message = error instanceof Error ? error.message : "";
      const problems = parseValidation(message);
      if (problems) {
        const lines = problems.map(({ field, code }) => {
          const label = fieldLabel(field);
          const reason = errors.has(code) ? errors(code) : errors("invalid");
          return `${label}: ${reason}`;
        });
        return `${errors("validationFailed")}\n${lines.join("\n")}`;
      }
      return errors.has(message) ? errors(message) : errors("generic");
    },
    [errors, fieldLabel]
  );
}
