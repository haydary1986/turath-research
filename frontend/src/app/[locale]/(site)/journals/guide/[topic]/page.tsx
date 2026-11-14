import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { getTranslations } from "next-intl/server";
import { SearchCheck } from "lucide-react";
import { Link } from "@/i18n/routing";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { ArticleBody } from "@/components/journals/article-body";
import { Breadcrumbs } from "@/components/journals/breadcrumbs";
import { JournalLinks } from "@/components/journals/journal-links";
import { JsonLd } from "@/components/seo/json-ld";
import { ARTICLES_UPDATED, getArticle, isTopicSlug } from "@/lib/journals/articles";
import { JOURNAL_PATHS } from "@/lib/journals/lists";
import { formatDate } from "@/lib/portal/types";
import { articleLd } from "@/lib/seo/json-ld";
import { pageMetadata } from "@/lib/seo/site";

type Props = { params: Promise<{ locale: string; topic: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale, topic } = await params;
  if (!isTopicSlug(topic)) return {};
  const article = getArticle(locale, topic);
  const t = await getTranslations({ locale, namespace: "journals" });
  return pageMetadata({
    locale, path: JOURNAL_PATHS.topic(topic), title: article.title, description: article.description,
    siteName: t("siteName"), type: "article",
  });
}

/** One topic article of the journal guide. */
export default async function TopicPage({ params }: Props) {
  const { locale, topic } = await params;
  if (!isTopicSlug(topic)) notFound();
  const article = getArticle(locale, topic);
  const t = await getTranslations({ locale, namespace: "journals" });
  const path = JOURNAL_PATHS.topic(topic);

  return (
    <article className="mx-auto max-w-4xl px-4 py-10">
      <Breadcrumbs
        locale={locale}
        crumbs={[
          { name: t("crumbs.hub"), path: JOURNAL_PATHS.hub },
          { name: t("crumbs.guide"), path: JOURNAL_PATHS.guide },
          { name: article.navLabel, path },
        ]}
      />
      <header className="mt-4">
        <h1 className="text-2xl font-bold text-primary text-balance md:text-4xl">{article.title}</h1>
        <p className="mt-4 text-lg leading-loose text-foreground/90">{article.intro}</p>
        <p className="mt-2 text-xs text-muted-foreground">{t("guide.lastUpdated", { date: formatDate(ARTICLES_UPDATED, locale) })}</p>
      </header>
      <div className="mt-10">
        <ArticleBody article={article} tocLabel={t("guide.onThisPage")} />
      </div>
      <div className="mt-10 flex flex-wrap items-center gap-3 rounded-xl border bg-muted/40 p-4">
        <Button nativeButton={false} render={<Link href={JOURNAL_PATHS.check} />}>
          <SearchCheck className="size-4" aria-hidden />
          {t("crumbs.check")}
        </Button>
        <Button variant="outline" nativeButton={false} render={<Link href={JOURNAL_PATHS.guide} />}>{t("crumbs.guide")}</Button>
      </div>
      <Alert variant="warning" className="mt-6">{t("checkPage.advisory")}</Alert>
      <JournalLinks locale={locale} current={path} heading={t("guide.topicsTitle")} />
      <JsonLd
        data={articleLd({
          locale, path, headline: article.title, description: article.description,
          publisher: t("siteName"), dateModified: ARTICLES_UPDATED,
        })}
      />
    </article>
  );
}
