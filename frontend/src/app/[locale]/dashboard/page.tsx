import { useTranslations } from "next-intl";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Users, FileText, Quote, TrendingUp } from "lucide-react";

export default function DashboardPage() {
  const t = useTranslations("dashboard");

  const stats = [
    { key: "totalResearchers", icon: Users, color: "text-blue-600" },
    { key: "totalPublications", icon: FileText, color: "text-green-600" },
    { key: "totalCitations", icon: Quote, color: "text-purple-600" },
    { key: "avgHIndex", icon: TrendingUp, color: "text-orange-600" },
  ] as const;

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t("title")}</h1>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {stats.map((stat) => (
          <Card key={stat.key}>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">
                {t(stat.key)}
              </CardTitle>
              <stat.icon className={`size-4 ${stat.color}`} />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">—</div>
            </CardContent>
          </Card>
        ))}
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>{t("recentPublications")}</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">
              {t("recentPlaceholder")}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{t("topResearchers")}</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">
              {t("topPlaceholder")}
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
