import { useTranslations } from "next-intl";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Users, FileText, Quote, TrendingUp } from "lucide-react";

export default function DashboardPage() {
  const t = useTranslations("dashboard");

  const stats = [
    {
      title: t("totalResearchers"),
      value: "—",
      icon: Users,
      color: "text-blue-600",
    },
    {
      title: t("totalPublications"),
      value: "—",
      icon: FileText,
      color: "text-green-600",
    },
    {
      title: t("totalCitations"),
      value: "—",
      icon: Quote,
      color: "text-purple-600",
    },
    {
      title: t("avgHIndex"),
      value: "—",
      icon: TrendingUp,
      color: "text-orange-600",
    },
  ];

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t("title")}</h1>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {stats.map((stat) => (
          <Card key={stat.title}>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">
                {stat.title}
              </CardTitle>
              <stat.icon className={`size-4 ${stat.color}`} />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stat.value}</div>
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
              سيتم عرض آخر المنشورات هنا بعد استيراد البيانات
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{t("topResearchers")}</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">
              سيتم عرض أفضل الباحثين هنا بعد استيراد البيانات
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
