"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { Link } from "@/i18n/routing";
import { api } from "@/lib/api";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Search } from "lucide-react";
import { Bidi, Ltr } from "@/components/i18n/bidi";

interface Publication {
  id: string;
  doi?: string;
  title_ar?: string;
  title_en?: string;
  publication_date?: string;
  publication_type: string;
  status: string;
  citation_count: number;
}

const PUBLICATION_TYPES = ["journal_article", "conference_paper", "book_chapter", "book", "preprint", "thesis", "patent", "report"];

export default function PublicationsPage() {
  const t = useTranslations("publication");
  const ct = useTranslations("common");
  const [pubs, setPubs] = useState<Publication[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchPubs();
  }, [page, search]);

  async function fetchPubs() {
    setLoading(true);
    try {
      const params = new URLSearchParams({
        page: String(page),
        limit: "20",
        ...(search ? { q: search } : {}),
      });
      const res = await api.get<Publication[]>(`/api/v1/publications?${params}`);
      setPubs(res.data || []);
      setTotal(res.meta?.total || 0);
    } catch {
      setPubs([]);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">{t("title")}</h1>
        <Badge variant="secondary">{total} {t("title")}</Badge>
      </div>

      <Card>
        <CardHeader>
          <div className="relative">
            <Search className="absolute start-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder={ct("search") + "..."}
              value={search}
              onChange={(e) => {
                setSearch(e.target.value);
                setPage(1);
              }}
              className="ps-10"
            />
          </div>
        </CardHeader>
        <CardContent>
          {loading ? (
            <p className="text-center text-muted-foreground">{ct("loading")}</p>
          ) : pubs.length === 0 ? (
            <p className="text-center text-muted-foreground">{ct("noResults")}</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[40%]">{t("titleAr")}</TableHead>
                  <TableHead>{t("type")}</TableHead>
                  <TableHead>{t("year")}</TableHead>
                  <TableHead>{t("citations")}</TableHead>
                  <TableHead>{t("doi")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {pubs.map((p) => (
                  <TableRow key={p.id}>
                    <TableCell>
                      <Link
                        href={`/publications/${p.id}`}
                        className="font-medium hover:underline"
                      >
                        <Bidi text={p.title_ar || p.title_en} fallback="—" />
                      </Link>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline">
                        {PUBLICATION_TYPES.includes(p.publication_type) ? t(`types.${p.publication_type}`) : <Bidi text={p.publication_type} />}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      {p.publication_date ? <Ltr>{new Date(p.publication_date).getFullYear()}</Ltr> : "—"}
                    </TableCell>
                    <TableCell>{p.citation_count}</TableCell>
                    <TableCell className="text-xs">
                      <Ltr>{p.doi || "—"}</Ltr>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}

          {total > 20 && (
            <div className="mt-4 flex items-center justify-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage((prev) => Math.max(1, prev - 1))}
                disabled={page === 1}
              >
                {ct("previous")}
              </Button>
              <span className="text-sm">{page}</span>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage((prev) => prev + 1)}
                disabled={page * 20 >= total}
              >
                {ct("next")}
              </Button>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
