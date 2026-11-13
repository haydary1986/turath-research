import type { Metadata } from "next";
import { getTranslations } from "next-intl/server";
import { NewsDetail } from "@/components/site/news-detail";

type Props = { params: Promise<{ locale: string; id: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "site.meta" });
  return { title: t("announcementTitle"), description: t("announcementDescription") };
}

export default async function AnnouncementPage({ params }: Props) {
  const { id } = await params;
  return (
    <div className="mx-auto max-w-3xl px-4 py-10">
      <NewsDetail id={id} />
    </div>
  );
}
