import type { Metadata } from "next";
import { getTranslations } from "next-intl/server";
import { LandingPage } from "@/components/site/landing-page";

type Props = { params: Promise<{ locale: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "site.meta" });
  return { title: t("homeTitle"), description: t("description") };
}

/** The locale root (/ar, /en) is the public face of the scientific affairs department. */
export default function HomePage() {
  return <LandingPage />;
}
