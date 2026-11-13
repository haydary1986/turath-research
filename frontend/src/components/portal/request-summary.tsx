"use client";

import { useLocale, useTranslations } from "next-intl";
import { fieldsFor } from "@/lib/portal/fields";
import { formatDate, type PortalRequest } from "@/lib/portal/types";

interface Row {
  label: string;
  value: string;
  href?: string;
  ltr?: boolean;
}

function isSafeUrl(value: string): boolean {
  return value.startsWith("https://") || value.startsWith("http://");
}

/** Key/value listing of everything the applicant entered; shared by the detail page and the printable form. */
export function RequestSummary({ request }: { request: PortalRequest }) {
  const t = useTranslations("portal");
  const locale = useLocale();

  const rows: Row[] = [
    { label: t("fields.requestNo"), value: request.request_no ?? "—", ltr: true },
    { label: t("fields.applicant"), value: request.applicant_name },
    { label: t("fields.college"), value: request.college_name ?? "—" },
    { label: t("fields.department"), value: request.department_name ?? "—" },
    { label: t("fields.title_ar"), value: request.title_ar ?? "—" },
    { label: t("fields.title_en"), value: request.title_en ?? "—", ltr: true },
    ...(request.kind === "research_plan"
      ? []
      : [{ label: t("fields.doi"), value: request.doi ?? "—", ltr: true }]),
    { label: t("fields.submittedAt"), value: formatDate(request.submitted_at, locale), ltr: true },
  ];

  for (const field of fieldsFor(request.kind)) {
    const value = request.details[field.key];
    if (!value) continue;
    const shown =
      field.type === "select" && t.has(`options.${value}`)
        ? t(`options.${value}`)
        : field.type === "date"
          ? formatDate(value, locale)
          : value;
    rows.push({
      label: t(`fields.${field.key}`),
      value: shown,
      href: field.type === "url" && isSafeUrl(value) ? value : undefined,
      ltr: field.ltr || field.type === "date",
    });
  }

  return (
    <dl className="grid gap-x-6 gap-y-3 text-sm md:grid-cols-2">
      {rows.map((row) => (
        <div key={row.label} className="min-w-0">
          <dt className="text-xs text-muted-foreground">{row.label}</dt>
          <dd className="break-words font-medium" dir={row.ltr ? "ltr" : undefined}>
            {row.href ? (
              <a href={row.href} target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">
                {row.value}
              </a>
            ) : (
              <span className="whitespace-pre-line">{row.value}</span>
            )}
          </dd>
        </div>
      ))}
    </dl>
  );
}
