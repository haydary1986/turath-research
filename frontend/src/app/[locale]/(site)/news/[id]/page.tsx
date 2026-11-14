import type { Metadata } from "next";
import { getTranslations } from "next-intl/server";
import { NewsDetail } from "@/components/site/news-detail";
import { metaDescription, pageMetadata } from "@/lib/seo/site";
import { excerpt } from "@/lib/site/rich-text";
import { REVALIDATE, backendGetOptional } from "@/lib/site/server-api";
import { PUBLIC_API, type PublicAnnouncement } from "@/lib/site/types";

type Props = { params: Promise<{ locale: string; id: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale, id } = await params;
  const [t, found] = await Promise.all([
    getTranslations({ locale, namespace: "site.meta" }),
    backendGetOptional<PublicAnnouncement>(PUBLIC_API.announcement(id), REVALIDATE.content),
  ]);
  const item = found?.data;
  // An announcement is written in one language and shown as written on both locales.
  const summary = item?.body ? metaDescription(excerpt(item.body, 300)) : "";
  return pageMetadata({
    locale,
    path: `/news/${encodeURIComponent(id)}`,
    title: item?.title || t("announcementTitle"),
    description: summary || t("announcementDescription"),
    siteName: t("appName"),
    type: "article",
  });
}

export default async function AnnouncementPage({ params }: Props) {
  const { id } = await params;
  return (
    <div className="mx-auto max-w-3xl px-4 py-10">
      <NewsDetail id={id} />
    </div>
  );
}
