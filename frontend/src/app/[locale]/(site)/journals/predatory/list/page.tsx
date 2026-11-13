import type { Metadata } from "next";
import { getTranslations } from "next-intl/server";
import { Breadcrumbs } from "@/components/journals/breadcrumbs";
import { JournalLinks } from "@/components/journals/journal-links";
import { LetterNav } from "@/components/journals/letter-nav";
import { ListOverview } from "@/components/journals/list-overview";
import { PublicListSearch } from "@/components/journals/public-list-search";
import { JsonLd } from "@/components/seo/json-ld";
import { Alert } from "@/components/ui/alert";
import { loadLetters, loadListStatuses } from "@/lib/journals/data";
import { JOURNAL_PATHS, LIST_TYPES } from "@/lib/journals/lists";
import { itemListLd } from "@/lib/seo/json-ld";
import { pageMetadata } from "@/lib/seo/site";

type Props = { params: Promise<{ locale: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "journals" });
  return pageMetadata({
    locale, path: JOURNAL_PATHS.list, title: t("listPage.title"), description: t("listPage.description"),
    siteName: t("siteName"),
  });
}

/** The lists hub: sizes, A–Z links to every index (server-rendered) and the interactive search. */
export default async function ListsPage({ params }: Props) {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "journals" });
  const [statuses, ...letters] = await Promise.all([
    loadListStatuses(),
    ...LIST_TYPES.map((type) => loadLetters(type).catch(() => [])),
  ]);

  return (
    <div className="mx-auto max-w-5xl px-4 py-10">
      <Breadcrumbs
        locale={locale}
        crumbs={[
          { name: t("crumbs.hub"), path: JOURNAL_PATHS.hub },
          { name: t("crumbs.list"), path: JOURNAL_PATHS.list },
        ]}
      />
      <h1 className="mt-4 text-2xl font-bold text-primary text-balance md:text-4xl">{t("listPage.title")}</h1>
      <p className="mt-3 max-w-3xl text-lg leading-relaxed text-muted-foreground">{t("listPage.intro")}</p>

      <div className="mt-8">
        <ListOverview locale={locale} statuses={statuses} />
      </div>

      <section aria-labelledby="browse-title" className="mt-10 space-y-6">
        <h2 id="browse-title" className="text-xl font-bold text-primary md:text-2xl">{t("listPage.browse")}</h2>
        {LIST_TYPES.map((type, i) => (
          <div key={type} className="space-y-2">
            <h3 className="font-semibold">{t(`lists.${type}`)}</h3>
            <LetterNav
              type={type}
              letters={letters[i]}
              label={`${t(`lists.${type}`)} — ${t("index.lettersLabel")}`}
              allLabel={t("index.all")}
              otherLabel={t("index.other")}
            />
          </div>
        ))}
      </section>

      <section aria-labelledby="search-title" className="mt-12 space-y-4">
        <h2 id="search-title" className="text-xl font-bold text-primary md:text-2xl">{t("listPage.searchTitle")}</h2>
        <PublicListSearch />
        <Alert variant="warning">{t("entry.disclaimer")}</Alert>
      </section>

      <JournalLinks locale={locale} current={JOURNAL_PATHS.list} heading={t("guide.topicsTitle")} />
      <JsonLd
        data={itemListLd(
          locale,
          t("listPage.title"),
          LIST_TYPES.map((type) => ({ name: t(`lists.${type}`), path: JOURNAL_PATHS.index(type) }))
        )}
      />
    </div>
  );
}
