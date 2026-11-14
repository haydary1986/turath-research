import type { Metadata } from "next";
import { getTranslations } from "next-intl/server";
import { ExternalLink, SearchCheck } from "lucide-react";
import { Link } from "@/i18n/routing";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { ArticleBody } from "@/components/journals/article-body";
import { Breadcrumbs } from "@/components/journals/breadcrumbs";
import { JournalLinks } from "@/components/journals/journal-links";
import { ListOverview } from "@/components/journals/list-overview";
import { JsonLd } from "@/components/seo/json-ld";
import { FaqList } from "@/components/site/faq-list";
import { ARTICLES_UPDATED, GUIDE_KEY, getArticle } from "@/lib/journals/articles";
import { loadListStatuses } from "@/lib/journals/data";
import { JOURNAL_PATHS } from "@/lib/journals/lists";
import { articleLd, faqLd } from "@/lib/seo/json-ld";
import { pageMetadata } from "@/lib/seo/site";
import { formatDate } from "@/lib/portal/types";

type Props = { params: Promise<{ locale: string }> };

const SOURCE_LINKS = [
  { key: "home", href: "https://beallslist.net/" },
  { key: "recognize", href: "https://beallslist.net/how-to-recognize-predatory-journals/" },
  { key: "metrics", href: "https://beallslist.net/misleading-metrics/" },
  { key: "pjHome", href: "https://www.predatoryjournals.org/home" },
] as const;

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const article = getArticle(locale, GUIDE_KEY);
  const t = await getTranslations({ locale, namespace: "journals" });
  return pageMetadata({
    locale, path: JOURNAL_PATHS.guide, title: article.title, description: article.description,
    siteName: t("siteName"), type: "article",
  });
}

/** The public predatory-journals guide: what they are, why they matter, the lists and an FAQ. */
export default async function PredatoryGuidePage({ params }: Props) {
  const { locale } = await params;
  const article = getArticle(locale, GUIDE_KEY);
  const [t, tp, statuses] = await Promise.all([
    getTranslations({ locale, namespace: "journals" }),
    getTranslations({ locale, namespace: "portal.predatory" }),
    loadListStatuses(),
  ]);

  return (
    <article className="mx-auto max-w-4xl px-4 py-10">
      <Breadcrumbs
        locale={locale}
        crumbs={[
          { name: t("crumbs.hub"), path: JOURNAL_PATHS.hub },
          { name: t("crumbs.guide"), path: JOURNAL_PATHS.guide },
        ]}
      />
      <header className="mt-4">
        <h1 className="text-2xl font-bold text-primary text-balance md:text-4xl">{article.title}</h1>
        <p className="mt-4 text-lg leading-loose text-foreground/90">{article.intro}</p>
        <p className="mt-2 text-xs text-muted-foreground">{t("guide.lastUpdated", { date: formatDate(ARTICLES_UPDATED, locale) })}</p>
        <Button className="mt-5" nativeButton={false} render={<Link href={JOURNAL_PATHS.check} />}>
          <SearchCheck className="size-4" aria-hidden />
          {t("crumbs.check")}
        </Button>
      </header>

      <section aria-labelledby="lists-title" className="mt-10">
        <h2 id="lists-title" className="mb-4 text-xl font-bold text-primary md:text-2xl">{t("listPage.searchTitle")}</h2>
        <ListOverview locale={locale} statuses={statuses} />
      </section>

      <div className="mt-10">
        <ArticleBody article={article} tocLabel={t("guide.onThisPage")} />
      </div>

      {!!article.faq?.length && (
        <section aria-labelledby="faq-title" className="mt-12">
          <h2 id="faq-title" className="mb-4 text-xl font-bold text-primary md:text-2xl">{t("guide.faqTitle")}</h2>
          <FaqList items={article.faq} />
          <JsonLd data={faqLd(article.faq)} />
        </section>
      )}

      <section aria-labelledby="sources-title" className="mt-12 space-y-3">
        <h2 id="sources-title" className="text-xl font-bold text-primary">{t("guide.sourcesTitle")}</h2>
        <ul className="flex flex-wrap gap-2">
          {SOURCE_LINKS.map((l) => (
            <li key={l.key}>
              <Button variant="outline" size="sm" nativeButton={false} render={<a href={l.href} target="_blank" rel="noopener noreferrer" />}>
                <ExternalLink className="size-4" aria-hidden />
                {tp(`links.${l.key}`)}
              </Button>
            </li>
          ))}
        </ul>
        <Alert variant="warning">{tp("disclaimer")}</Alert>
      </section>

      <JournalLinks locale={locale} current={JOURNAL_PATHS.guide} heading={t("guide.topicsTitle")} />
      <JsonLd
        data={articleLd({
          locale, path: JOURNAL_PATHS.guide, headline: article.title, description: article.description,
          publisher: t("siteName"), dateModified: ARTICLES_UPDATED,
        })}
      />
    </article>
  );
}
