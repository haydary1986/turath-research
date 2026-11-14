"use client";

import { useTranslations } from "next-intl";
import { ExternalLink } from "lucide-react";
import { REFERENCE_GROUPS } from "@/lib/journals/reference-links";

/** Grouped outbound links to the predatory, hijacked and ministry journal lists. */
export function ReferenceLinks() {
  const t = useTranslations("journals.refLinks");

  return (
    <div className="grid gap-4 md:grid-cols-3">
      {REFERENCE_GROUPS.map((group) => (
        <section key={group.key} aria-labelledby={`ref-${group.key}`} className="rounded-xl border bg-card p-5">
          <h2 id={`ref-${group.key}`} className="text-lg font-bold text-primary">{t(`groups.${group.key}.title`)}</h2>
          <p className="mt-1 text-sm text-muted-foreground">{t(`groups.${group.key}.source`)}</p>
          <ul className="mt-4 space-y-2">
            {group.links.map((link) => (
              <li key={link.key}>
                <a
                  href={link.href}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center justify-between gap-2 rounded-lg border px-3 py-2.5 font-medium text-primary hover:border-accent/40 hover:text-accent"
                >
                  <span>{t(`links.${link.key}`)}</span>
                  <ExternalLink className="size-4 shrink-0" aria-hidden />
                </a>
              </li>
            ))}
          </ul>
        </section>
      ))}
    </div>
  );
}
