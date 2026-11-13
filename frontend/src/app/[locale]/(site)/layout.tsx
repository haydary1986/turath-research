import type { ReactNode } from "react";
import { getTranslations } from "next-intl/server";
import { JsonLd } from "@/components/seo/json-ld";
import { SiteHeader } from "@/components/site/site-header";
import { SiteFooter } from "@/components/site/site-footer";
import { organizationLd } from "@/lib/seo/json-ld";
import { loadSiteContent } from "@/lib/site/public-data";

type Props = {
  children: ReactNode;
  params: Promise<{ locale: string }>;
};

/** Public site shell (landing page, circulars and activities, journal guide): no app sidebar. */
export default async function SiteLayout({ children, params }: Props) {
  const { locale } = await params;
  const [content, t, ta] = await Promise.all([
    loadSiteContent(),
    getTranslations({ locale, namespace: "site.meta" }),
    getTranslations({ locale, namespace: "auth" }),
  ]);

  return (
    <div className="flex min-h-svh flex-col">
      <SiteHeader />
      <main id="main" tabIndex={-1} className="flex-1 outline-none">
        {children}
      </main>
      <SiteFooter locale={locale} content={content} />
      <JsonLd
        data={organizationLd(locale, {
          university: ta("universityName"),
          department: t("homeTitle"),
          description: t("description"),
        })}
      />
    </div>
  );
}
