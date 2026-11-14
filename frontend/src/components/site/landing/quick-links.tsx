import { getTranslations } from "next-intl/server";
import { ExternalLink } from "lucide-react";
import { Localized } from "@/components/i18n/localized";
import type { SiteLink } from "@/lib/site/types";
import { Section } from "./section";

/** Only https links leave the site; anything else an editor typed is not rendered. */
function isSafeExternal(url: string): boolean {
  return url.startsWith("https://");
}

/** Important external sites the department picks in the site editor (e.g. the ministry's Nadwati platform). */
export async function QuickLinks({ locale, links }: { locale: string; links: SiteLink[] }) {
  const safe = links.filter((link) => isSafeExternal(link.url));
  if (safe.length === 0) return null;
  const t = await getTranslations({ locale, namespace: "landing.quickLinks" });

  return (
    <Section id="links" title={t("title")}>
      <ul className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        {safe.map((link, i) => (
          <li key={i}>
            <a
              href={link.url}
              target="_blank"
              rel="noopener noreferrer"
              className="group flex h-full items-center justify-between gap-3 rounded-xl border bg-card px-4 py-3.5 font-semibold text-primary transition-shadow hover:border-accent/40 hover:shadow-md focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
            >
              <Localized ar={link.label} en={link.label_en} locale={locale} />
              <ExternalLink className="size-4 shrink-0 text-accent" aria-hidden />
              <span className="sr-only">{t("newTab")}</span>
            </a>
          </li>
        ))}
      </ul>
    </Section>
  );
}
