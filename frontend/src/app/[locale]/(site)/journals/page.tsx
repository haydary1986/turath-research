import type { Metadata } from "next";
import { getTranslations } from "next-intl/server";
import { Breadcrumbs } from "@/components/journals/breadcrumbs";
import { JournalLinks } from "@/components/journals/journal-links";
import { JsonLd } from "@/components/seo/json-ld";
import { articlesFor, GUIDE_KEY, TOPIC_SLUGS } from "@/lib/journals/articles";
import { JOURNAL_PATHS } from "@/lib/journals/lists";
import { itemListLd } from "@/lib/seo/json-ld";
import { pageMetadata } from "@/lib/seo/site";

type Props = { params: Promise<{ locale: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "journals" });
  return pageMetadata({
    locale, path: JOURNAL_PATHS.hub, title: t("hub.title"), description: t("hub.description"), siteName: t("siteName"),
  });
}

/** Hub of the public journal guide: every topic and tool in one place. */
export default async function JournalsHubPage({ params }: Props) {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "journals" });
  const articles = articlesFor(locale);
  const items = [
    { name: articles[GUIDE_KEY].title, path: JOURNAL_PATHS.guide },
    ...TOPIC_SLUGS.map((slug) => ({ name: articles[slug].title, path: JOURNAL_PATHS.topic(slug) })),
    { name: t("crumbs.check"), path: JOURNAL_PATHS.check },
  ];

  return (
    <div className="mx-auto max-w-5xl px-4 py-10">
      <Breadcrumbs locale={locale} crumbs={[{ name: t("crumbs.hub"), path: JOURNAL_PATHS.hub }]} />
      <h1 className="mt-4 text-2xl font-bold text-primary text-balance md:text-4xl">{t("hub.title")}</h1>
      <p className="mt-3 max-w-3xl text-lg leading-relaxed text-muted-foreground">{t("hub.intro")}</p>
      <JournalLinks locale={locale} current={JOURNAL_PATHS.hub} heading={t("hub.topicsTitle")} />
      <JsonLd data={itemListLd(locale, t("hub.title"), items)} />
    </div>
  );
}
