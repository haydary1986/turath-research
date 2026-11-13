import type { ReactNode } from "react";
import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { getTranslations } from "next-intl/server";
import { ExternalLink, Flag, SearchCheck } from "lucide-react";
import { Link } from "@/i18n/routing";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Breadcrumbs } from "@/components/journals/breadcrumbs";
import { EntryLinks } from "@/components/journals/entry-links";
import { loadEntry } from "@/lib/journals/data";
import { JOURNAL_PATHS, SOURCE_LIST_URL, type JournalEntryDetail } from "@/lib/journals/lists";
import { loadSiteContent } from "@/lib/site/public-data";
import { formatDate } from "@/lib/portal/types";
import { absoluteUrl, metaDescription, pageMetadata } from "@/lib/seo/site";

type Props = { params: Promise<{ locale: string; slug: string }> };

function refreshedAt(detail: JournalEntryDetail): string {
  return detail.list_refreshed_at ?? detail.entry.fetched_at;
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale, slug } = await params;
  const detail = await loadEntry(slug);
  if (!detail) return {};
  const t = await getTranslations({ locale, namespace: "journals" });
  const { entry } = detail;
  const list = t(`lists.${entry.list_type}`);
  return pageMetadata({
    locale,
    path: JOURNAL_PATHS.entry(entry.slug),
    title: t("entry.metaTitle", { name: entry.name, list }),
    description: metaDescription(
      t("entry.metaDescription", { name: entry.name, list, date: formatDate(refreshedAt(detail), locale) })
    ),
    siteName: t("siteName"),
  });
}

/** A prefilled email to the department, or the footer's contact block when no email is published. */
function correctionHref(locale: string, email: string | undefined, subject: string, body: string): string {
  if (!email) return `/${locale}#contact`;
  return `mailto:${email}?subject=${encodeURIComponent(subject)}&body=${encodeURIComponent(body)}`;
}

/**
 * One listed journal or publisher. Worded as what the source list says (with its
 * date), never as the university's verdict, with a visible correction request.
 */
export default async function EntryPage({ params }: Props) {
  const { locale, slug } = await params;
  const [detail, content, t] = await Promise.all([
    loadEntry(slug),
    loadSiteContent(),
    getTranslations({ locale, namespace: "journals" }),
  ]);
  if (!detail) notFound();

  const { entry } = detail;
  const list = t(`lists.${entry.list_type}`);
  const hijacked = entry.list_type === "hijacked";
  const pageUrl = absoluteUrl(locale, JOURNAL_PATHS.entry(entry.slug));
  const correction = correctionHref(
    locale,
    content?.contact.email || undefined,
    t("entry.correctionSubject", { name: entry.name }),
    t("entry.correctionBody", { url: pageUrl })
  );
  const facts: { label: string; value: ReactNode }[] = [
    { label: t("entry.listedName"), value: <span dir="ltr">{entry.name}</span> },
    { label: t("entry.listType"), value: list },
    {
      label: hijacked ? t("entry.fakeHost") : t("entry.host"),
      value: entry.hosts.length ? <span dir="ltr" className={hijacked ? "text-status-danger-fg" : undefined}>{entry.hosts.join(" · ")}</span> : t("entry.noHost"),
    },
  ];
  if (hijacked && entry.authentic_name) {
    facts.push({ label: t("entry.authenticJournal"), value: <span dir="ltr">{entry.authentic_name}</span> });
  }
  if (hijacked && entry.authentic_url) {
    facts.push({
      label: t("entry.authenticSite"),
      value: (
        <a href={entry.authentic_url} target="_blank" rel="noopener noreferrer nofollow" dir="ltr"
          className="inline-flex items-center gap-1 break-all text-status-success-fg hover:underline">
          {entry.authentic_url}
          <ExternalLink className="size-3.5 shrink-0" aria-hidden />
        </a>
      ),
    });
  }
  facts.push(
    {
      label: t("entry.source"),
      value: (
        <a href={SOURCE_LIST_URL} target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">
          Stop Predatory Journals
        </a>
      ),
    },
    { label: t("entry.lastRefresh"), value: formatDate(refreshedAt(detail), locale) }
  );

  return (
    <article className="mx-auto max-w-4xl px-4 py-10">
      <Breadcrumbs
        locale={locale}
        crumbs={[
          { name: t("crumbs.hub"), path: JOURNAL_PATHS.hub },
          { name: t("crumbs.list"), path: JOURNAL_PATHS.list },
          { name: list, path: JOURNAL_PATHS.index(entry.list_type) },
          { name: entry.name, path: JOURNAL_PATHS.entry(entry.slug) },
        ]}
      />
      <header className="mt-4">
        <p className="text-sm font-semibold text-accent">{list}</p>
        <h1 className="mt-1 text-2xl font-bold break-words text-primary md:text-3xl" dir="auto">{entry.name}</h1>
        <p className="mt-4 text-lg leading-relaxed">{t(`entry.statement.${entry.list_type}`, { name: entry.name })}</p>
      </header>

      <section aria-labelledby="facts-title" className="mt-8 rounded-xl border bg-card">
        <h2 id="facts-title" className="border-b px-4 py-3 font-bold text-primary">{t("entry.factsTitle")}</h2>
        <dl className="divide-y">
          {facts.map((fact) => (
            <div key={fact.label} className="grid gap-1 px-4 py-3 sm:grid-cols-[14rem_1fr]">
              <dt className="text-sm text-muted-foreground">{fact.label}</dt>
              <dd className="min-w-0 font-medium break-words">{fact.value}</dd>
            </div>
          ))}
        </dl>
      </section>

      <section aria-labelledby="meaning-title" className="mt-8">
        <h2 id="meaning-title" className="text-xl font-bold text-primary">{t("entry.meaningTitle")}</h2>
        <p className="mt-2 leading-loose">{t(`entry.meaning.${entry.list_type}`)}</p>
        <h2 className="mt-6 text-xl font-bold text-primary">{t("entry.whatToDoTitle")}</h2>
        <p className="mt-2 leading-loose">{t("entry.whatToDo")}</p>
        <Button className="mt-4" render={<Link href={`${JOURNAL_PATHS.check}?q=${encodeURIComponent(entry.name)}`} />}>
          <SearchCheck className="size-4" aria-hidden />
          {t("entry.checkCta")}
        </Button>
      </section>

      <Alert variant="warning" className="mt-8">
        <p>{t("entry.disclaimer")}</p>
        <p className="mt-2">
          {t("entry.correctionHint")}{" "}
          <a href={correction} className="inline-flex items-center gap-1 font-semibold underline underline-offset-4">
            <Flag className="size-3.5" aria-hidden />
            {t("entry.correction")}
          </a>
        </p>
      </Alert>

      <EntryLinks
        title={t("entry.sameHost")}
        id="same-host"
        links={detail.same_host}
        listLabel={(type) => t(`lists.${type}`)}
      />
      <EntryLinks
        title={t("entry.neighbours")}
        id="neighbours"
        links={detail.neighbours}
        listLabel={(type) => t(`lists.${type}`)}
      />

      <p className="mt-10">
        <Link href={JOURNAL_PATHS.index(entry.list_type)} className="font-semibold text-primary hover:text-accent hover:underline">
          {t("entry.backToIndex", { list })}
        </Link>
      </p>
    </article>
  );
}
