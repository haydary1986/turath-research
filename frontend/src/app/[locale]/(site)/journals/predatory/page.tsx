import type { Metadata } from "next";
import { getTranslations } from "next-intl/server";
import { SearchCheck } from "lucide-react";
import { Link } from "@/i18n/routing";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Breadcrumbs } from "@/components/journals/breadcrumbs";
import { JournalLinks } from "@/components/journals/journal-links";
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

/** The public predatory-journals page: links to the trusted lists, nothing copied from them. */
export default async function PredatoryLinksPage({ params }: Props) {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "journals" });

  return (
    <article className="mx-auto max-w-5xl px-4 py-10">
      <Breadcrumbs
        locale={locale}
        crumbs={[
          { name: t("crumbs.hub"), path: JOURNAL_PATHS.hub },
          { name: t("crumbs.guide"), path: JOURNAL_PATHS.guide },
        ]}
      />
      <header className="mt-4">
        <h1 className="text-2xl font-bold text-primary text-balance md:text-4xl">{t("refLinks.title")}</h1>
        <p className="mt-3 max-w-3xl text-lg leading-relaxed text-muted-foreground">{t("refLinks.intro")}</p>
      </header>

      <div className="mt-8">
        <ReferenceLinks />
      </div>

      <div className="mt-8 flex flex-wrap items-center gap-3 rounded-xl border bg-muted/40 p-4">
        <p className="flex-1 text-sm text-muted-foreground">{t("refLinks.checkHint")}</p>
        <Button nativeButton={false} render={<Link href={JOURNAL_PATHS.check} />}>
          <SearchCheck className="size-4" aria-hidden />
          {t("crumbs.check")}
        </Button>
      </div>
      <Alert variant="warning" className="mt-6">{t("refLinks.disclaimer")}</Alert>

      <JournalLinks locale={locale} current={JOURNAL_PATHS.guide} heading={t("guide.topicsTitle")} />
    </article>
  );
}
