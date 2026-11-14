"use client";

import { Bidi, Ltr } from "@/components/i18n/bidi";
import { isLink } from "@/lib/portal/ris-sync";

/** One synced value: a link opens in a new tab, an empty value shows a dash. */
export function ValueCell({ value, departments }: { value: string | null; departments: Map<string, string> }) {
  if (!value) return <span className="text-muted-foreground">—</span>;
  if (isLink(value)) {
    return (
      <a href={value} target="_blank" rel="noopener noreferrer" className="break-all text-primary hover:underline" dir="ltr">
        {value.replace(/^https:\/\/(www\.)?/, "")}
      </a>
    );
  }
  const department = departments.get(value);
  if (department) return <Bidi text={department} />;
  return /^[\d\s+-]+$|^\d{4}-\d{4}/.test(value) ? <Ltr>{value}</Ltr> : <Bidi text={value} />;
}
