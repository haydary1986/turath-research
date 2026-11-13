import type { Metadata } from "next";
import { getTranslations } from "next-intl/server";
import { NewsList } from "@/components/site/news-list";
import { isAnnouncementType } from "@/lib/site/types";

type Props = {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ type?: string | string[]; page?: string | string[] }>;
};

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "site.meta" });
  return { title: t("newsTitle"), description: t("newsDescription") };
}

function single(value: string | string[] | undefined): string | undefined {
  return Array.isArray(value) ? value[0] : value;
}

export default async function NewsPage({ params, searchParams }: Props) {
  const { locale } = await params;
  const query = await searchParams;
  const t = await getTranslations({ locale, namespace: "site" });
  const type = single(query.type);
  const page = Number.parseInt(single(query.page) ?? "1", 10);

  return (
    <div className="mx-auto max-w-4xl px-4 py-10">
      <h1 className="border-s-4 border-accent ps-3 text-2xl font-bold text-primary md:text-3xl">{t("meta.newsTitle")}</h1>
      <p className="mt-2 mb-8 text-muted-foreground">{t("news.intro")}</p>
      <NewsList
        initialType={isAnnouncementType(type) ? type : null}
        initialPage={Number.isFinite(page) && page > 0 ? page : 1}
      />
    </div>
  );
}
