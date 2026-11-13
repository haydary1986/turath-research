import type { Metadata } from "next";
import { getTranslations } from "next-intl/server";
import { LandingPage } from "@/components/site/landing-page";
import { pageMetadata } from "@/lib/seo/site";
import { loadLanding } from "@/lib/site/public-data";

type Props = { params: Promise<{ locale: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "site.meta" });
  return pageMetadata({
    locale, path: "/", title: t("appName"), description: t("description"), siteName: t("appName"), absoluteTitle: true,
  });
}

/** The locale root (/ar, /en) is the public face of the scientific affairs department. */
export default async function HomePage({ params }: Props) {
  const { locale } = await params;
  const data = await loadLanding();
  return <LandingPage locale={locale} data={data} />;
}
