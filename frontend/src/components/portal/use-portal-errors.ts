"use client";

import { useCallback } from "react";
import { useTranslations } from "next-intl";
import { parseValidation } from "@/lib/portal/types";

/** Turns an API error (plain code or "validation:field:code,...") into a readable localized message. */
export function usePortalErrors(): (error: unknown) => string {
  const errors = useTranslations("portal.errors");
  const fields = useTranslations("portal.fields");
  const roles = useTranslations("portal.roles");

  return useCallback(
    (error: unknown): string => {
      const message = error instanceof Error ? error.message : "";

      const problems = parseValidation(message);
      if (problems) {
        const lines = problems.map(({ field, code }) => {
          const label =
            field === "attachment"
              ? roles.has(code) ? roles(code) : code
              : fields.has(field) ? fields(field) : field;
          const reason =
            field === "attachment"
              ? errors("required")
              : errors.has(code) ? errors(code) : errors("invalid");
          return `${label}: ${reason}`;
        });
        return `${errors("validationFailed")}\n${lines.join("\n")}`;
      }

      return errors.has(message) ? errors(message) : errors("generic");
    },
    [errors, fields, roles]
  );
}
