"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { Link } from "@/i18n/routing";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Plus, Search } from "lucide-react";

interface Researcher {
  id: string;
  name_ar: string;
  name_en?: string;
  email: string;
  title_ar?: string;
  h_index: number;
  total_citations: number;
  total_publications: number;
  status: string;
}

export default function ResearchersPage() {
  const t = useTranslations("researcher");
  const ct = useTranslations("common");
  const { token } = useAuth();
  const [researchers, setResearchers] = useState<Researcher[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchResearchers();
  }, [page, search]);

  async function fetchResearchers() {
    setLoading(true);
    try {
      const params = new URLSearchParams({
        page: String(page),
        limit: "20",
        ...(search ? { q: search } : {}),
      });
      const res = await api.get<Researcher[]>(
        `/api/v1/researchers?${params}`
      );
      setResearchers(res.data || []);
      setTotal(res.meta?.total || 0);
    } catch {
      setResearchers([]);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">{t("title")}</h1>
        <Button render={<Link href="/researchers/new" />}>
          <Plus className="size-4" />
          <span>{t("addNew")}</span>
        </Button>
      </div>

      <Card>
        <CardHeader>
          <div className="flex items-center gap-4">
            <div className="relative flex-1">
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
            <Badge variant="secondary">{total} {t("title")}</Badge>
          </div>
        </CardHeader>
        <CardContent>
          {loading ? (
            <p className="text-center text-muted-foreground">{ct("loading")}</p>
          ) : researchers.length === 0 ? (
            <p className="text-center text-muted-foreground">{ct("noResults")}</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("name")}</TableHead>
                  <TableHead>{t("email")}</TableHead>
                  <TableHead>{t("academicTitle")}</TableHead>
                  <TableHead>{t("hIndex")}</TableHead>
                  <TableHead>{t("citations")}</TableHead>
                  <TableHead>{t("publications")}</TableHead>
                  <TableHead>{t("status")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {researchers.map((r) => (
                  <TableRow key={r.id}>
                    <TableCell>
                      <Link
                        href={`/researchers/${r.id}`}
                        className="font-medium hover:underline"
                      >
                        {r.name_ar}
                        {r.name_en && (
                          <span className="ms-2 text-xs text-muted-foreground">
                            {r.name_en}
                          </span>
                        )}
                      </Link>
                    </TableCell>
                    <TableCell dir="ltr">{r.email}</TableCell>
                    <TableCell>{r.title_ar || "—"}</TableCell>
                    <TableCell>{r.h_index}</TableCell>
                    <TableCell>{r.total_citations}</TableCell>
                    <TableCell>{r.total_publications}</TableCell>
                    <TableCell>
                      <Badge
                        variant={r.status === "active" ? "default" : "secondary"}
                      >
                        {r.status === "active" ? t("active") : t("inactive")}
                      </Badge>
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
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page === 1}
              >
                {ct("previous")}
              </Button>
              <span className="text-sm">{page}</span>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage((p) => p + 1)}
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
