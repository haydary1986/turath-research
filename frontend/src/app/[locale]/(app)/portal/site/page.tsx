"use client";

import { useSearchParams } from "next/navigation";
import { Suspense } from "react";
import { useTranslations } from "next-intl";
import { ExternalLink } from "lucide-react";
import { Link } from "@/i18n/routing";
import { useAuth } from "@/lib/auth";
import { canManageSite } from "@/lib/site/types";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { RequireAuth } from "@/components/portal/require-auth";
import { AnnouncementsManager } from "@/components/site-admin/announcements-manager";
import { SiteContentForm } from "@/components/site-admin/site-content-form";

function SiteAdmin() {
  const t = useTranslations("portal.site");
  const role = useAuth((state) => state.user?.role);
  const editId = useSearchParams().get("edit") ?? undefined;

  if (!canManageSite(role)) {
    return <Alert variant="danger">{t("forbidden")}</Alert>;
  }

  return (
    <div className="mx-auto max-w-4xl space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold">{t("title")}</h1>
          <p className="text-sm text-muted-foreground">{t("intro")}</p>
        </div>
        <Button variant="outline" render={<Link href="/" target="_blank" />}>
          <ExternalLink aria-hidden />
          {t("viewSite")}
        </Button>
      </div>
      <Tabs defaultValue={editId ? "announcements" : "content"}>
        <TabsList>
          <TabsTrigger value="content">{t("tabs.content")}</TabsTrigger>
          <TabsTrigger value="announcements">{t("tabs.announcements")}</TabsTrigger>
        </TabsList>
        <TabsContent value="content" className="pt-4">
          <SiteContentForm />
        </TabsContent>
        <TabsContent value="announcements" className="pt-4">
          <AnnouncementsManager initialEditId={editId} />
        </TabsContent>
      </Tabs>
    </div>
  );
}

export default function SiteAdminPage() {
  return (
    <RequireAuth>
      <Suspense>
        <SiteAdmin />
      </Suspense>
    </RequireAuth>
  );
}
