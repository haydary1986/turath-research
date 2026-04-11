"use client";

import { useLocale, useTranslations } from "next-intl";
import { useRouter, usePathname } from "@/i18n/routing";
import { SidebarTrigger } from "@/components/ui/sidebar";
import { Button } from "@/components/ui/button";
import { Languages } from "lucide-react";

export function AppHeader() {
  const t = useTranslations("common");
  const locale = useLocale();
  const router = useRouter();
  const pathname = usePathname();

  function toggleLocale() {
    const next = locale === "ar" ? "en" : "ar";
    router.replace(pathname, { locale: next });
  }

  return (
    <header className="flex h-14 items-center gap-4 border-b bg-background px-6">
      <SidebarTrigger />
      <div className="flex-1" />
      <span className="text-sm font-medium">{t("appName")}</span>
      <div className="flex-1" />
      <Button variant="ghost" size="sm" onClick={toggleLocale}>
        <Languages className="size-4" />
        <span className="ms-2 text-xs">
          {locale === "ar" ? "English" : "العربية"}
        </span>
      </Button>
    </header>
  );
}
