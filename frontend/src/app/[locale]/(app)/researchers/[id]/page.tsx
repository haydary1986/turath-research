"use client";

import { useEffect, useState, use } from "react";
import { useTranslations } from "next-intl";
import { api } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { FileText, Quote, TrendingUp } from "lucide-react";
import { ResearcherContactCard } from "@/components/portal/researcher-contact-card";
import { ResearcherLinksCard } from "@/components/portal/researcher-links-card";
import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
  PieChart, Pie, Cell,
} from "recharts";
import { Bidi, Ltr } from "@/components/i18n/bidi";

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

type Props = { params: Promise<{ id: string }> };

export default function ResearcherProfilePage({ params }: Props) {
  const { id } = use(params);
  const t = useTranslations("researcher");
  const ct = useTranslations("common");
  const dt = useTranslations("dashboard");
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
    { key: "hIndex", value: stats.h_index, icon: TrendingUp, color: "text-primary", bg: "bg-primary/10" },
    { key: "citations", value: stats.total_citations, icon: Quote, color: "text-chart-3", bg: "bg-chart-3/10" },
    { key: "publications", value: stats.total_publications, icon: FileText, color: "text-accent", bg: "bg-accent/10" },
  ] as const;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <Bidi as="h1" block text={stats.name_ar} className="text-2xl font-bold" />
      </div>

      <ResearcherContactCard researcherId={id} />
      <ResearcherLinksCard researcherId={id} />

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
            <CardTitle className="text-sm">{t("byYear")}</CardTitle>
          </CardHeader>
          <CardContent>
            {stats.publications_by_year.length === 0 ? (
              <p className="text-sm text-muted-foreground">{t("noData")}</p>
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
            <CardTitle className="text-sm">{t("rankingDistribution")}</CardTitle>
          </CardHeader>
          <CardContent>
            {stats.journal_distribution.length === 0 ? (
              <p className="text-sm text-muted-foreground">{t("noData")}</p>
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
                      <span>{d.ranking === "unranked" ? t("unranked") : <Ltr>{d.ranking}</Ltr>}</span>
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
          <CardTitle className="text-sm">{t("topCited")}</CardTitle>
        </CardHeader>
        <CardContent>
          {stats.top_publications.length === 0 ? (
            <p className="text-sm text-muted-foreground">{t("noData")}</p>
          ) : (
            <div className="space-y-3">
              {stats.top_publications.map((p, i) => (
                <div key={p.id} className="flex items-start gap-3">
                  <span className="mt-0.5 flex size-6 shrink-0 items-center justify-center rounded-full bg-muted text-xs font-bold">
                    {i + 1}
                  </span>
                  <div className="flex-1">
                    <Bidi as="p" block text={p.title} className="text-sm" />
                    <div className="mt-1 flex gap-3 text-xs text-muted-foreground">
                      {p.year && <Ltr>{p.year}</Ltr>}
                      {p.doi && <Ltr>{p.doi}</Ltr>}
                    </div>
                  </div>
                  <Badge variant="secondary" className="shrink-0">
                    {dt("metrics.citations", { count: p.citation_count })}
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
            <CardTitle className="text-sm">{t("coAuthors")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-2">
              {stats.co_authors.map((ca) => (
                <Badge key={ca.name} variant="outline">
                  <Bidi text={ca.name} /> (<Ltr>{ca.papers}</Ltr>)
                </Badge>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
