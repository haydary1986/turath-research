import type { Metadata } from "next";
import { getTranslations } from "next-intl/server";
import { Alert } from "@/components/ui/alert";
import { Breadcrumbs } from "@/components/journals/breadcrumbs";
import { JournalLinks } from "@/components/journals/journal-links";
import { PublicJournalCheck } from "@/components/journals/public-journal-check";
import { JOURNAL_PATHS } from "@/lib/journals/lists";
import { pageMetadata } from "@/lib/seo/site";

type Props = {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ q?: string | string[] }>;
};

const MAX_PREFILL = 300;

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "journals" });
  return pageMetadata({
    locale, path: JOURNAL_PATHS.check, title: t("checkPage.title"), description: t("checkPage.description"),
    siteName: t("siteName"),
  });
}

/** Public journal check: server-rendered explanation, interactive search on top. */
export default async function JournalCheckPage({ params, searchParams }: Props) {
  const { locale } = await params;
  const { q } = await searchParams;
  const prefill = (Array.isArray(q) ? q[0] : q)?.slice(0, MAX_PREFILL) ?? "";
  const t = await getTranslations({ locale, namespace: "journals" });

  return (
    <div className="mx-auto max-w-4xl px-4 py-10">
      <Breadcrumbs
        locale={locale}
        crumbs={[
          { name: t("crumbs.hub"), path: JOURNAL_PATHS.hub },
          { name: t("crumbs.check"), path: JOURNAL_PATHS.check },
        ]}
      />
      <h1 className="mt-4 text-2xl font-bold text-primary text-balance md:text-4xl">{t("checkPage.title")}</h1>
      <p className="mt-3 text-lg leading-relaxed text-muted-foreground">{t("checkPage.intro")}</p>

      <div className="mt-8">
        <PublicJournalCheck initialText={prefill} />
      </div>

      <section aria-labelledby="how-title" className="mt-10">
        <h2 id="how-title" className="text-xl font-bold text-primary">{t("checkPage.howTitle")}</h2>
        <ul className="mt-3 space-y-2 leading-relaxed">
          {(["how1", "how2", "how3", "how4"] as const).map((key) => (
            <li key={key} className="flex gap-2.5">
              <span className="mt-2.5 size-1.5 shrink-0 rounded-full bg-accent" aria-hidden />
              <span>{t(`checkPage.${key}`)}</span>
            </li>
          ))}
        </ul>
        <Alert variant="warning" className="mt-4">{t("checkPage.advisory")}</Alert>
      </section>

      <JournalLinks locale={locale} current={JOURNAL_PATHS.check} heading={t("guide.topicsTitle")} />
    </div>
  );
}
