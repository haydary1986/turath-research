import type { Metadata } from "next";
import { getTranslations } from "next-intl/server";
import { NewsDetail } from "@/components/site/news-detail";
import { pageAlternates } from "@/lib/seo/site";

type Props = { params: Promise<{ locale: string; id: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale, id } = await params;
  const t = await getTranslations({ locale, namespace: "site.meta" });
  return {
    title: t("announcementTitle"),
    description: t("announcementDescription"),
    alternates: pageAlternates(locale, `/news/${encodeURIComponent(id)}`),
  };
}

export default async function AnnouncementPage({ params }: Props) {
  const { id } = await params;
  return (
    <div className="mx-auto max-w-3xl px-4 py-10">
      <NewsDetail id={id} />
    </div>
  );
}
