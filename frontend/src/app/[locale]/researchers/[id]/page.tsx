"use client";

import { useEffect, useState, use } from "react";
import { useTranslations } from "next-intl";
import { api } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { FileText, Quote, TrendingUp } from "lucide-react";
import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
  PieChart, Pie, Cell,
} from "recharts";

interface ResearcherStats {
  id: string;
  name_ar: string;
  h_index: number;
  total_citations: number;
  total_publications: number;
  publications_by_year: { year: number; count: number }[];
  journal_distribution: { ranking: string; count: number }[];
  top_publications: { id: string; title: string; doi?: string; citation_count: number; year?: number }[];
  co_authors: { id?: string; name: string; papers: number }[];
}

const COLORS = ["#2563eb", "#16a34a", "#eab308", "#dc2626", "#6b7280"];
const RANKING_LABELS: Record<string, string> = {
  Q1: "Q1", Q2: "Q2", Q3: "Q3", Q4: "Q4", unranked: "غير مصنف",
};

type Props = { params: Promise<{ id: string }> };

export default function ResearcherProfilePage({ params }: Props) {
  const { id } = use(params);
  const t = useTranslations("researcher");
  const ct = useTranslations("common");
  const [stats, setStats] = useState<ResearcherStats | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.get<ResearcherStats>(`/api/v1/metrics/researcher/${id}`)
      .then((res) => setStats(res.data || null))
      .catch(() => setStats(null))
      .finally(() => setLoading(false));
  }, [id]);

  if (loading) return <p className="text-center py-10">{ct("loading")}</p>;
  if (!stats) return <p className="text-center py-10">{ct("noResults")}</p>;

  const kpiCards = [
    { key: "hIndex", value: stats.h_index, icon: TrendingUp, color: "text-blue-600", bg: "bg-blue-50" },
    { key: "citations", value: stats.total_citations, icon: Quote, color: "text-purple-600", bg: "bg-purple-50" },
    { key: "publications", value: stats.total_publications, icon: FileText, color: "text-green-600", bg: "bg-green-50" },
  ] as const;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold">{stats.name_ar}</h1>
      </div>

      {/* KPI Cards */}
      <div className="grid gap-4 md:grid-cols-3">
        {kpiCards.map((s) => (
          <Card key={s.key}>
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm text-muted-foreground">{t(s.key)}</CardTitle>
              <div className={`rounded-lg p-2 ${s.bg}`}>
                <s.icon className={`size-4 ${s.color}`} />
              </div>
            </CardHeader>
            <CardContent>
              <div className="text-3xl font-bold">{s.value}</div>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Charts Row */}
      <div className="grid gap-4 md:grid-cols-2">
        {/* Publications by Year */}
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">الأبحاث حسب السنة</CardTitle>
          </CardHeader>
          <CardContent>
            {stats.publications_by_year.length === 0 ? (
              <p className="text-sm text-muted-foreground">لا توجد بيانات</p>
            ) : (
              <ResponsiveContainer width="100%" height={250}>
                <BarChart data={stats.publications_by_year}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="year" fontSize={12} />
                  <YAxis fontSize={12} />
                  <Tooltip />
                  <Bar dataKey="count" fill="#2563eb" radius={[4, 4, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            )}
          </CardContent>
        </Card>

        {/* Journal Distribution */}
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">توزيع تصنيف المجلات</CardTitle>
          </CardHeader>
          <CardContent>
            {stats.journal_distribution.length === 0 ? (
              <p className="text-sm text-muted-foreground">لا توجد بيانات</p>
            ) : (
              <div className="flex items-center gap-6">
                <ResponsiveContainer width="50%" height={200}>
                  <PieChart>
                    <Pie
                      data={stats.journal_distribution}
                      dataKey="count"
                      nameKey="ranking"
                      cx="50%"
                      cy="50%"
                      outerRadius={80}
                    >
                      {stats.journal_distribution.map((_, i) => (
                        <Cell key={i} fill={COLORS[i % COLORS.length]} />
                      ))}
                    </Pie>
                    <Tooltip />
                  </PieChart>
                </ResponsiveContainer>
                <div className="space-y-2">
                  {stats.journal_distribution.map((d, i) => (
                    <div key={d.ranking} className="flex items-center gap-2 text-sm">
                      <span
                        className="size-3 rounded-full"
                        style={{ backgroundColor: COLORS[i % COLORS.length] }}
                      />
                      <span>{RANKING_LABELS[d.ranking] || d.ranking}</span>
                      <span className="font-bold">{d.count}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Top Publications */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">أكثر الأبحاث استشهاداً</CardTitle>
        </CardHeader>
        <CardContent>
          {stats.top_publications.length === 0 ? (
            <p className="text-sm text-muted-foreground">لا توجد بيانات</p>
          ) : (
            <div className="space-y-3">
              {stats.top_publications.map((p, i) => (
                <div key={p.id} className="flex items-start gap-3">
                  <span className="mt-0.5 flex size-6 shrink-0 items-center justify-center rounded-full bg-muted text-xs font-bold">
                    {i + 1}
                  </span>
                  <div className="flex-1">
                    <p className="text-sm">{p.title}</p>
                    <div className="mt-1 flex gap-3 text-xs text-muted-foreground">
                      {p.year && <span>{p.year}</span>}
                      {p.doi && <span dir="ltr">{p.doi}</span>}
                    </div>
                  </div>
                  <Badge variant="secondary" className="shrink-0">
                    {p.citation_count} cit
                  </Badge>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Co-authors */}
      {stats.co_authors.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">المؤلفون المشاركون</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-2">
              {stats.co_authors.map((ca) => (
                <Badge key={ca.name} variant="outline">
                  {ca.name} ({ca.papers})
                </Badge>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
