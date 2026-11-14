"use client";

import { useLocale, useTranslations } from "next-intl";
import { ExternalLink } from "lucide-react";
import { Link } from "@/i18n/routing";
import { Badge } from "@/components/ui/badge";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Bidi, Ltr } from "@/components/i18n/bidi";

export interface PublicationAuthor {
  name: string;
  name_en?: string | null;
  researcher_id: string | null;
}

export interface PublicationItem {
  id: string;
  doi?: string | null;
  title_ar?: string | null;
  title_en?: string | null;
  publication_date?: string | null;
  publication_type: string;
  citation_count: number;
  journal_name?: string | null;
  authors: PublicationAuthor[];
  authors_count: number;
}

function AuthorList({ authors, total, locale }: { authors: PublicationAuthor[]; total: number; locale: string }) {
  const t = useTranslations("publication.filters");
  if (authors.length === 0) return <span className="text-muted-foreground">—</span>;
  return (
    <span className="flex flex-wrap gap-x-2 gap-y-1">
      {authors.map((a, i) => {
        const name = locale === "en" && a.name_en ? a.name_en : a.name;
        return a.researcher_id ? (
          <Link key={i} href={`/researchers/${a.researcher_id}`} className="text-primary hover:underline">
            <Bidi text={name} />
          </Link>
        ) : (
          <span key={i}><Bidi text={name} /></span>
        );
      })}
      {total > authors.length && (
        <span className="text-muted-foreground">{t("moreAuthors", { count: total - authors.length })}</span>
      )}
    </span>
  );
}

/** The publications list: title with venue, the university's authors first, year, type, citations and DOI. */
export function PublicationsTable({ items }: { items: PublicationItem[] }) {
  const t = useTranslations("publication");
  const locale = useLocale();

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="w-[42%]">{t("filters.titleColumn")}</TableHead>
          <TableHead>{t("authors")}</TableHead>
          <TableHead>{t("year")}</TableHead>
          <TableHead>{t("type")}</TableHead>
          <TableHead className="text-end">{t("citations")}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {items.map((p) => {
          const title = locale === "en" ? p.title_en || p.title_ar : p.title_ar || p.title_en;
          return (
            <TableRow key={p.id} className="align-top">
              <TableCell className="whitespace-normal">
                <Link href={`/publications/${p.id}`} className="font-medium leading-snug hover:underline">
                  <Bidi text={title} fallback="—" />
                </Link>
                <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
                  {p.journal_name && <Bidi text={p.journal_name} />}
                  {p.doi && (
                    <a href={`https://doi.org/${p.doi}`} target="_blank" rel="noopener noreferrer"
                      className="inline-flex items-center gap-1 hover:text-primary hover:underline">
                      <Ltr>{p.doi}</Ltr>
                      <ExternalLink className="size-3" aria-hidden />
                    </a>
                  )}
                </div>
              </TableCell>
              <TableCell className="whitespace-normal text-sm">
                <AuthorList authors={p.authors} total={p.authors_count} locale={locale} />
              </TableCell>
              <TableCell>{p.publication_date ? <Ltr>{p.publication_date.slice(0, 4)}</Ltr> : "—"}</TableCell>
              <TableCell>
                <Badge variant="outline">
                  {t.has(`types.${p.publication_type}`) ? t(`types.${p.publication_type}`) : p.publication_type}
                </Badge>
              </TableCell>
              <TableCell className="text-end tabular-nums">{p.citation_count}</TableCell>
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
}
