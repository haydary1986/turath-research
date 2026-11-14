"use client";

import { useTranslations } from "next-intl";
import { ClipboardCheck } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { RequireAuth } from "@/components/portal/require-auth";
import { Link } from "@/i18n/routing";
import { useMyResponsibilities } from "@/lib/portal/my-access";

/** «مسؤولياتي»: the categories the signed-in user is responsible for entering. */
function MyResponsibilities() {
  const tr = useTranslations("portal.responsibilities");
  const ta = useTranslations("portal.activities");
  const responsibilities = useMyResponsibilities();

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{tr("title")}</h1>
        <p className="text-sm text-muted-foreground">{tr("intro")}</p>
      </div>
      {responsibilities.length === 0 ? (
        <Card>
          <CardContent className="py-8 text-center text-sm text-muted-foreground">{tr("none")}</CardContent>
        </Card>
      ) : (
        <div className="grid gap-3 sm:grid-cols-2">
          {responsibilities.map((r) => (
            <Card key={r.id ?? `${r.category}-${r.college_id ?? "u"}`}>
              <CardHeader>
                <CardTitle className="flex items-start gap-2 text-base">
                  <ClipboardCheck className="mt-0.5 size-5 shrink-0 text-primary" />
                  <span>{ta(`categories.${r.category}`)}</span>
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <div className="flex flex-wrap gap-2">
                  <Badge variant="secondary">{r.college_name ?? tr("university")}</Badge>
                </div>
                <Button size="sm" variant="outline"
                  nativeButton={false} render={<Link href={`/portal/activities?category=${r.category}`} />}>
                  {tr("viewRecords")}
                </Button>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
      <p className="text-sm text-muted-foreground">{ta("fs003.hint")}</p>
    </div>
  );
}

export default function Page() {
  return (
    <RequireAuth>
      <MyResponsibilities />
    </RequireAuth>
  );
}
