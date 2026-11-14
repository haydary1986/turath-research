"use client";

import { useLocale, useTranslations } from "next-intl";
import { coResearchers, fieldsFor, hasJournalData } from "@/lib/portal/fields";
import { formatDate, type PortalRequest } from "@/lib/portal/types";
import { Bidi, Ltr } from "@/components/i18n/bidi";

interface Row {
  label: string;
  value: string;
  href?: string;
  ltr?: boolean;
  /** A translated label (an option name), not text the applicant typed. */
  chrome?: boolean;
}

function isSafeUrl(value: string): boolean {
  return value.startsWith("https://") || value.startsWith("http://");
}

/** Key/value listing of everything the applicant entered; shared by the detail page and the printable form. */
export function RequestSummary({ request }: { request: PortalRequest }) {
  const t = useTranslations("portal");
  const locale = useLocale();

  const isTitleChange = request.kind === "title_change";
  const rows: Row[] = [
    { label: t("fields.requestNo"), value: request.request_no ?? "—", ltr: true },
    { label: t("fields.applicant"), value: request.applicant_name },
    { label: t("fields.college"), value: request.college_name ?? "—" },
    { label: t("fields.department"), value: request.department_name ?? "—" },
    { label: t(isTitleChange ? "fields.new_title_ar" : "fields.title_ar"), value: request.title_ar ?? "—" },
    { label: t(isTitleChange ? "fields.new_title_en" : "fields.title_en"), value: request.title_en ?? "—" },
    ...(hasJournalData(request.kind)
      ? [{ label: t("fields.doi"), value: request.doi ?? "—", ltr: true }]
      : []),
    { label: t("fields.submittedAt"), value: formatDate(request.submitted_at, locale), ltr: true },
  ];

  for (const field of fieldsFor(request.kind)) {
    const value = request.details[field.key];
    if (!value) continue;
    const translated = field.type === "select" && t.has(`options.${value}`);
    const shown = translated ? t(`options.${value}`) : field.type === "date" ? formatDate(value, locale) : value;
    rows.push({
      label: t(`fields.${field.key}`),
      value: shown,
      href: field.type === "url" && isSafeUrl(value) ? value : undefined,
      ltr: field.ltr || field.type === "date" || field.type === "url",
      chrome: translated,
    });
  }

  if (request.kind === "research_plan") {
    const partners = coResearchers(request.details);
    // Plans saved before the structured fields kept the partners in one free-text field.
    const legacy = request.details.participants?.trim();
    if (partners.length > 0 || legacy) {
      rows.push({ label: t("fields.co_researchers"), value: partners.length > 0 ? partners.join("\n") : legacy });
    }
  }

  return (
    <dl className="grid gap-x-6 gap-y-3 text-sm md:grid-cols-2">
      {rows.map((row) => (
        <div key={row.label} className="min-w-0">
          <dt className="text-xs text-muted-foreground">{row.label}</dt>
          <dd className="break-words font-medium">
            {row.href ? (
              <a href={row.href} target="_blank" rel="noopener noreferrer" dir="ltr" data-bidi="" className="text-primary hover:underline">
                {row.value}
              </a>
            ) : row.chrome || row.value === "—" ? (
              <span>{row.value}</span>
            ) : row.ltr ? (
              <Ltr className="whitespace-pre-line">{row.value}</Ltr>
            ) : (
              <Bidi block text={row.value} className="whitespace-pre-line" />
            )}
          </dd>
        </div>
      ))}
    </dl>
  );
}
