import type { Metadata } from "next";
import { getTranslations } from "next-intl/server";
import { Alert } from "@/components/ui/alert";
import { Breadcrumbs } from "@/components/journals/breadcrumbs";
import { PredatoryExplainer } from "@/components/journals/predatory-explainer";
import { ReferenceLinks } from "@/components/journals/reference-links";
import { JOURNAL_PATHS } from "@/lib/journals/lists";
import { pageMetadata } from "@/lib/seo/site";

type Props = { params: Promise<{ locale: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "journals" });
  return pageMetadata({
    locale, path: JOURNAL_PATHS.guide, title: t("refLinks.title"), description: t("refLinks.intro"),
    siteName: t("siteName"),
  });
}

/** The public predatory-journals page: a short explanation, then links to the trusted lists. */
export default async function PredatoryJournalsPage({ params }: Props) {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "journals" });

  return (
    <article className="mx-auto max-w-5xl px-4 py-10">
      <Breadcrumbs locale={locale} crumbs={[{ name: t("crumbs.guide"), path: JOURNAL_PATHS.guide }]} />
      <header className="mt-4">
        <h1 className="text-2xl font-bold text-primary text-balance md:text-4xl">{t("refLinks.title")}</h1>
        <p className="mt-3 max-w-3xl text-lg leading-relaxed text-muted-foreground">{t("refLinks.intro")}</p>
      </header>

      <div className="mt-8">
        <PredatoryExplainer />
      </div>

      <h2 className="mt-10 mb-4 text-xl font-bold text-primary md:text-2xl">{t("refLinks.linksTitle")}</h2>
      <ReferenceLinks />
      <Alert variant="warning" className="mt-6">{t("refLinks.disclaimer")}</Alert>
    </article>
  );
}
