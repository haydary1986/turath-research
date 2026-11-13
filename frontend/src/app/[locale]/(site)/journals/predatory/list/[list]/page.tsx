import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { getTranslations } from "next-intl/server";
import { Link } from "@/i18n/routing";
import { Breadcrumbs } from "@/components/journals/breadcrumbs";
import { LetterNav } from "@/components/journals/letter-nav";
import { JsonLd } from "@/components/seo/json-ld";
import { loadIndexPage, loadLetters } from "@/lib/journals/data";
import {
  INDEX_PAGE_SIZE, JOURNAL_PATHS, LETTER_OTHER, isLetter, listTypeFromSegment, type ListType,
} from "@/lib/journals/lists";
import { formatDate } from "@/lib/portal/types";
import { itemListLd } from "@/lib/seo/json-ld";
import { pageMetadata } from "@/lib/seo/site";

type Props = {
  params: Promise<{ locale: string; list: string }>;
  searchParams: Promise<{ letter?: string | string[]; page?: string | string[] }>;
};

function single(value: string | string[] | undefined): string | undefined {
  return Array.isArray(value) ? value[0] : value;
}

interface IndexQuery {
  type: ListType;
  letter: string | undefined;
  page: number;
}

async function readQuery(props: Props): Promise<IndexQuery | null> {
  const [{ list }, query] = await Promise.all([props.params, props.searchParams]);
  const type = listTypeFromSegment(list);
  if (!type) return null;
  const rawLetter = single(query.letter)?.toLowerCase();
  const letter = isLetter(rawLetter) ? rawLetter : undefined;
  const page = Math.max(1, Number.parseInt(single(query.page) ?? "1", 10) || 1);
  return { type, letter, page };
}

async function indexTitle(locale: string, { type, letter }: IndexQuery): Promise<string> {
  const t = await getTranslations({ locale, namespace: "journals" });
  const list = t(`lists.${type}`);
  if (!letter) return t("index.title", { list });
  if (letter === LETTER_OTHER) return t("index.titleOther", { list });
  return t("index.titleLetter", { list, letter: letter.toUpperCase() });
}

export async function generateMetadata(props: Props): Promise<Metadata> {
  const { locale } = await props.params;
  const q = await readQuery(props);
  if (!q) return {};
  const t = await getTranslations({ locale, namespace: "journals" });
  const title = await indexTitle(locale, q);
  return pageMetadata({
    locale,
    path: JOURNAL_PATHS.index(q.type, q.letter, q.page),
    // Each page of a letter gets its own title so no two index pages share one.
    title: q.page > 1 ? `${title} (${q.page})` : title,
    description: t("index.description", { list: t(`lists.${q.type}`) }),
    siteName: t("siteName"),
  });
}

/** Server-rendered A–Z index of one list, paginated with plain links. */
export default async function ListIndexPage(props: Props) {
  const { locale } = await props.params;
  const q = await readQuery(props);
  if (!q) notFound();
  const [t, letters, result] = await Promise.all([
    getTranslations({ locale, namespace: "journals" }),
    loadLetters(q.type),
    loadIndexPage(q.type, q.letter, q.page),
  ]);
  const total = result.meta?.total ?? result.data.length;
  const pages = Math.max(1, Math.ceil(total / INDEX_PAGE_SIZE));
  if (q.page > pages) notFound();
  const title = await indexTitle(locale, q);
  const offset = (q.page - 1) * INDEX_PAGE_SIZE;

  return (
    <div className="mx-auto max-w-5xl px-4 py-10">
      <Breadcrumbs
        locale={locale}
        crumbs={[
          { name: t("crumbs.hub"), path: JOURNAL_PATHS.hub },
          { name: t("crumbs.list"), path: JOURNAL_PATHS.list },
          { name: title, path: JOURNAL_PATHS.index(q.type, q.letter, q.page) },
        ]}
      />
      <h1 className="mt-4 text-2xl font-bold text-primary md:text-3xl">{title}</h1>
      <p className="mt-2 text-muted-foreground">{t(`listHelp.${q.type}`)}</p>
      <p className="mt-1 text-sm text-muted-foreground tabular-nums">{t("listPage.count", { count: total })}</p>

      <div className="mt-6">
        <LetterNav
          type={q.type}
          letters={letters}
          label={t("index.lettersLabel")}
          allLabel={t("index.all")}
          otherLabel={t("index.other")}
          current={q.letter}
        />
      </div>

      {result.data.length === 0 ? (
        <p className="mt-8 rounded-xl border border-dashed p-6 text-center text-muted-foreground">{t("index.empty")}</p>
      ) : (
        <ol start={offset + 1} className="mt-8 divide-y rounded-xl border bg-card">
          {result.data.map((entry) => (
            <li key={entry.slug} className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1 px-4 py-3">
              <Link
                href={JOURNAL_PATHS.entry(entry.slug)}
                dir="ltr"
                className="min-w-0 font-medium text-primary hover:text-accent hover:underline"
              >
                {entry.name}
              </Link>
              <span className="text-xs text-muted-foreground" dir="ltr">
                {entry.hosts.join(" · ")}
                {entry.hosts.length > 0 && " · "}
                {formatDate(entry.fetched_at, locale)}
              </span>
            </li>
          ))}
        </ol>
      )}

      {pages > 1 && (
        <nav aria-label={t("index.pagination")} className="mt-6 flex flex-wrap items-center justify-between gap-3 text-sm">
          {q.page > 1 ? (
            <Link rel="prev" href={JOURNAL_PATHS.index(q.type, q.letter, q.page - 1)} className="rounded-lg border px-3 py-2 hover:border-accent hover:text-accent">
              {t("index.previous")}
            </Link>
          ) : <span />}
          <span className="tabular-nums text-muted-foreground">{t("index.pageOf", { page: q.page, pages })}</span>
          {q.page < pages ? (
            <Link rel="next" href={JOURNAL_PATHS.index(q.type, q.letter, q.page + 1)} className="rounded-lg border px-3 py-2 hover:border-accent hover:text-accent">
              {t("index.next")}
            </Link>
          ) : <span />}
        </nav>
      )}

      <JsonLd
        data={itemListLd(
          locale,
          title,
          result.data.map((entry) => ({ name: entry.name, path: JOURNAL_PATHS.entry(entry.slug) })),
          offset
        )}
      />
    </div>
  );
}
