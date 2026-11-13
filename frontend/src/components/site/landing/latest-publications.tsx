import { getTranslations } from "next-intl/server";
import { BookMarked, ExternalLink } from "lucide-react";
import { formatNumber } from "@/lib/site/format";
import type { LocalizedName, PublicPublication } from "@/lib/site/types";
import { Section } from "./section";

function localized(name: LocalizedName, locale: string): string {
  return locale === "en" && name.en ? name.en : name.ar;
}

/** Title in the page language when available, else the other one. */
function titleOf(p: PublicPublication, locale: string): { text: string; lang: string } {
  if (locale === "en") return p.title_en ? { text: p.title_en, lang: "en" } : { text: p.title_ar ?? "", lang: "ar" };
  return p.title_ar ? { text: p.title_ar, lang: "ar" } : { text: p.title_en ?? "", lang: "en" };
}

/** «أحدث البحوث المنشورة لتدريسيي الجامعة»: newest papers with journal, year, college and DOI. */
export async function LatestPublications({ locale, items }: { locale: string; items: PublicPublication[] }) {
  if (items.length === 0) return null;
  const t = await getTranslations({ locale, namespace: "landing.publications" });

  return (
    <Section id="publications" title={t("title")}>
      <ul className="grid gap-4 md:grid-cols-2">
        {items.map((p) => {
          const title = titleOf(p, locale);
          return (
            <li key={p.id} className="flex flex-col rounded-2xl border bg-card p-5">
              <div className="flex items-start gap-3">
                <BookMarked className="mt-1 size-5 shrink-0 text-accent" aria-hidden />
                <h3 lang={title.lang} dir="auto" className="line-clamp-3 font-bold leading-snug text-primary">
                  {title.text}
                </h3>
              </div>
              <p className="mt-3 text-sm text-foreground/90">
                {localized(p.author, locale)}
                {p.college && <span className="text-muted-foreground"> · {localized(p.college, locale)}</span>}
              </p>
              <dl className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-sm text-muted-foreground">
                {p.journal && (
                  <div className="flex min-w-0 gap-1">
                    <dt className="sr-only">{t("journal")}</dt>
                    <dd dir="auto" className="truncate italic">{p.journal}</dd>
                  </div>
                )}
                {p.year && (
                  <div className="flex gap-1">
                    <dt className="sr-only">{t("year")}</dt>
                    <dd className="tabular-nums">{formatNumber(p.year, locale, false)}</dd>
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
                  <span dir="ltr">DOI: {p.doi}</span>
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
