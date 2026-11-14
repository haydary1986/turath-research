import { getTranslations } from "next-intl/server";
import { BookMarked, ExternalLink } from "lucide-react";
import { formatNumber } from "@/lib/site/format";
import type { LocalizedName, PublicPublication } from "@/lib/site/types";
import { Bidi, Ltr } from "@/components/i18n/bidi";
import { pickName } from "@/lib/i18n/text-lang";
import { Section } from "./section";

function NameText({ name, locale }: { name: LocalizedName; locale: string }) {
  const picked = pickName(name.ar, name.en, locale);
  return picked ? <Bidi text={picked.text} lang={picked.lang} /> : null;
}

/** «أحدث البحوث المنشورة لتدريسيي الجامعة»: newest papers with journal, year, college and DOI. */
export async function LatestPublications({ locale, items }: { locale: string; items: PublicPublication[] }) {
  if (items.length === 0) return null;
  const t = await getTranslations({ locale, namespace: "landing.publications" });

  return (
    <Section id="publications" title={t("title")}>
      <ul className="grid gap-4 md:grid-cols-2">
        {items.map((p) => {
          const title = pickName(p.title_ar, p.title_en, locale);
          return (
            <li key={p.id} className="flex flex-col rounded-2xl border bg-card p-5">
              <div className="flex items-start gap-3">
                <BookMarked className="mt-1 size-5 shrink-0 text-accent" aria-hidden />
                <Bidi as="h3" block text={title?.text} lang={title?.lang} className="line-clamp-3 font-bold leading-snug text-primary" />
              </div>
              <p className="mt-3 text-sm text-foreground/90">
                <NameText name={p.author} locale={locale} />
                {p.college && <span className="text-muted-foreground"> · <NameText name={p.college} locale={locale} /></span>}
              </p>
              <dl className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-sm text-muted-foreground">
                {p.journal && (
                  <div className="flex min-w-0 gap-1">
                    <dt className="sr-only">{t("journal")}</dt>
                    <Bidi as="dd" text={p.journal} className="truncate italic" />
                  </div>
                )}
                {p.year && (
                  <div className="flex gap-1">
                    <dt className="sr-only">{t("year")}</dt>
                    <dd className="tabular-nums" dir="ltr" data-bidi="">{formatNumber(p.year, locale, false)}</dd>
                  </div>
                )}
              </dl>
              {p.doi && (
                <a
                  href={`https://doi.org/${encodeURI(p.doi)}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="mt-auto inline-flex w-fit items-center gap-1.5 pt-3 text-sm font-semibold text-primary hover:text-accent hover:underline focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
                >
                  <Ltr>DOI: {p.doi}</Ltr>
                  <ExternalLink className="size-3.5" aria-hidden />
                  <span className="sr-only">{t("openDoi")} {t("opensInNewTab")}</span>
                </a>
              )}
            </li>
          );
        })}
      </ul>
    </Section>
  );
}
