"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";
import { Users, FileText, Quote, TrendingUp, GraduationCap, BarChart3 } from "lucide-react";

interface KPIs {
  total_researchers: number;
  active_researchers: number;
  total_publications: number;
  total_citations: number;
  avg_h_index: number;
  q1_publications: number;
  q1_ratio: number;
  publications_this_year: number;
}

interface LeaderEntry {
  id: string;
  name_ar: string;
  name_en?: string;
  h_index: number;
  total_citations: number;
  total_publications: number;
}

export default function DashboardPage() {
  const t = useTranslations("dashboard");
  const ct = useTranslations("common");
  const [kpis, setKpis] = useState<KPIs | null>(null);
  const [leaders, setLeaders] = useState<LeaderEntry[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function load() {
      try {
        const [kpiRes, leaderRes] = await Promise.all([
          api.get<KPIs>("/api/v1/metrics/institution"),
          api.get<LeaderEntry[]>("/api/v1/metrics/leaderboard?sort=h_index"),
        ]);
        setKpis(kpiRes.data || null);
        setLeaders(leaderRes.data || []);
      } catch {
        // Will show placeholder values
      } finally {
        setLoading(false);
      }
    }
    load();
  }, []);

  const stats = [
    {
      key: "totalResearchers",
      value: kpis?.active_researchers ?? "—",
      icon: Users,
      color: "text-primary",
      bg: "bg-primary/10",
    },
    {
      key: "totalPublications",
      value: kpis?.total_publications ?? "—",
      icon: FileText,
      color: "text-accent",
      bg: "bg-accent/10",
    },
    {
      key: "totalCitations",
      value: kpis?.total_citations ?? "—",
      icon: Quote,
      color: "text-chart-3",
      bg: "bg-chart-3/10",
    },
    {
      key: "avgHIndex",
      value: kpis ? kpis.avg_h_index.toFixed(1) : "—",
      icon: TrendingUp,
      color: "text-chart-4",
      bg: "bg-chart-4/10",
    },
    {
      key: "q1Ratio",
      value: kpis ? `${kpis.q1_ratio.toFixed(1)}%` : "—",
      icon: GraduationCap,
      color: "text-chart-5",
      bg: "bg-chart-5/10",
    },
    {
      key: "recentPublications",
      value: kpis?.publications_this_year ?? "—",
      subLabel: new Date().getFullYear().toString(),
      icon: BarChart3,
      color: "text-primary",
      bg: "bg-primary/10",
    },
  ] as const;

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t("title")}</h1>

      {/* KPI Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6">
        {stats.map((stat) => (
          <Card key={stat.key}>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-xs font-medium text-muted-foreground">
                {t(stat.key)}
              </CardTitle>
              <div className={`rounded-lg p-2 ${stat.bg}`}>
                <stat.icon className={`size-4 ${stat.color}`} />
              </div>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {loading ? "..." : stat.value}
              </div>
              {"subLabel" in stat && stat.subLabel && (
                <p className="text-xs text-muted-foreground">{stat.subLabel}</p>
              )}
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Leaderboard + Recent */}
      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>{t("topResearchers")}</CardTitle>
          </CardHeader>
          <CardContent>
            {loading ? (
              <p className="text-sm text-muted-foreground">{ct("loading")}</p>
            ) : leaders.length === 0 ? (
              <p className="text-sm text-muted-foreground">{t("topPlaceholder")}</p>
            ) : (
              <div className="space-y-3">
                {leaders.slice(0, 10).map((r, i) => (
                  <div key={r.id} className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <span className="flex size-7 items-center justify-center rounded-full bg-muted text-xs font-bold">
                        {i + 1}
                      </span>
                      <div>
                        <p className="text-sm font-medium">{r.name_ar}</p>
                        {r.name_en && (
                          <p className="text-xs text-muted-foreground">{r.name_en}</p>
                        )}
                      </div>
                    </div>
                    <div className="flex gap-4 text-xs text-muted-foreground">
                      <span>h:{r.h_index}</span>
                      <span>{r.total_citations} cit</span>
                      <span>{r.total_publications} pub</span>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{t("recentPublications")}</CardTitle>
          </CardHeader>
          <CardContent>
            <RecentPublications />
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

function RecentPublications() {
  const ct = useTranslations("common");
  const dt = useTranslations("dashboard");
  const [pubs, setPubs] = useState<{ id: string; title_ar?: string; title_en?: string; citation_count: number; publication_date?: string }[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.get<typeof pubs>("/api/v1/publications?limit=10&page=1")
      .then((res) => setPubs(res.data || []))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <p className="text-sm text-muted-foreground">{ct("loading")}</p>;
  if (pubs.length === 0) return <p className="text-sm text-muted-foreground">{dt("recentPlaceholder")}</p>;

  return (
    <div className="space-y-3">
      {pubs.map((p) => (
        <div key={p.id} className="flex items-start justify-between gap-2">
          <p className="text-sm line-clamp-2">{p.title_ar || p.title_en || "—"}</p>
          <span className="shrink-0 text-xs text-muted-foreground">
            {p.citation_count} cit
          </span>
        </div>
      ))}
    </div>
  );
}
