"use client";

import { useEffect, useState, use } from "react";
import { useTranslations } from "next-intl";
import { api } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Users, FileText, Quote, TrendingUp } from "lucide-react";

interface Researcher {
  id: string;
  name_ar: string;
  name_en?: string;
  email: string;
  title_ar?: string;
  title_en?: string;
  h_index: number;
  total_citations: number;
  total_publications: number;
  status: string;
  department_id?: string;
}

type Props = {
  params: Promise<{ id: string }>;
};

export default function ResearcherProfilePage({ params }: Props) {
  const { id } = use(params);
  const t = useTranslations("researcher");
  const ct = useTranslations("common");
  const [researcher, setResearcher] = useState<Researcher | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function load() {
      try {
        const res = await api.get<Researcher>(`/api/v1/researchers/${id}`);
        setResearcher(res.data || null);
      } catch {
        setResearcher(null);
      } finally {
        setLoading(false);
      }
    }
    load();
  }, [id]);

  if (loading) return <p className="text-center">{ct("loading")}</p>;
  if (!researcher) return <p className="text-center">{ct("noResults")}</p>;

  const stats = [
    { key: "hIndex", value: researcher.h_index, icon: TrendingUp, color: "text-blue-600" },
    { key: "citations", value: researcher.total_citations, icon: Quote, color: "text-purple-600" },
    { key: "publications", value: researcher.total_publications, icon: FileText, color: "text-green-600" },
  ] as const;

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <div>
          <h1 className="text-2xl font-bold">{researcher.name_ar}</h1>
          {researcher.name_en && (
            <p className="text-muted-foreground">{researcher.name_en}</p>
          )}
        </div>
        <Badge variant={researcher.status === "active" ? "default" : "secondary"}>
          {researcher.status === "active" ? t("active") : t("inactive")}
        </Badge>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        {stats.map((s) => (
          <Card key={s.key}>
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm">{t(s.key)}</CardTitle>
              <s.icon className={`size-4 ${s.color}`} />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{s.value}</div>
            </CardContent>
          </Card>
        ))}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t("profile")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex justify-between">
            <span className="text-muted-foreground">{t("email")}</span>
            <span dir="ltr">{researcher.email}</span>
          </div>
          {researcher.title_ar && (
            <div className="flex justify-between">
              <span className="text-muted-foreground">{t("academicTitle")}</span>
              <span>{researcher.title_ar}</span>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
