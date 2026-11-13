"use client";

import { useCallback } from "react";
import { useTranslations } from "next-intl";
import { parseValidation } from "@/lib/portal/types";

/** Localises API errors from the site endpoints, labelling fields with the site editor's names. */
export function useSiteErrors(): (error: unknown) => string {
  const errors = useTranslations("portal.errors");
  const fields = useTranslations("portal.site.fields");
  const blockFields = useTranslations("siteEditor.fields");

  return useCallback(
    (error: unknown): string => {
      const message = error instanceof Error ? error.message : "";
      const problems = parseValidation(message);
      if (problems) {
        const lines = problems.map(({ field, code }) => {
          const label = fields.has(field) ? fields(field) : blockFields.has(field) ? blockFields(field) : field;
          const reason = errors.has(code) ? errors(code) : errors("invalid");
          return `${label}: ${reason}`;
        });
        return `${errors("validationFailed")}\n${lines.join("\n")}`;
      }
      return errors.has(message) ? errors(message) : errors("generic");
    },
    [errors, fields, blockFields]
  );
}
