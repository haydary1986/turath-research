"use client";

import { useLocale, useTranslations } from "next-intl";
import { useRouter, usePathname } from "@/i18n/routing";
import { SidebarTrigger } from "@/components/ui/sidebar";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Languages, LogOut } from "lucide-react";
import { NotificationsBell } from "@/components/layout/notifications-bell";
import { useAuth } from "@/lib/auth";
import { useHydrated } from "@/hooks/use-hydrated";

function initials(name: string): string {
  const parts = name.trim().split(/\s+/).slice(0, 2);
  return parts.map((p) => p.charAt(0)).join(" ") || "؟";
}

function UserMenu() {
  const t = useTranslations("common");
  const roles = useTranslations("portal.users.roleNames");
  const locale = useLocale();
  const router = useRouter();
  const user = useAuth((state) => state.user);
  const logout = useAuth((state) => state.logout);
  const mounted = useHydrated();

  if (!mounted || !user) return null;

  const name = (locale === "ar" ? user.name_ar : user.name_en) || user.name_ar || user.name_en || user.email;

  function handleLogout() {
    logout();
    router.push("/auth/login");
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        aria-label={t("accountMenu")}
        className="flex items-center gap-2 rounded-full outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        <span className="flex size-8 items-center justify-center rounded-full bg-primary text-xs font-bold text-primary-foreground">
          {initials(name)}
        </span>
        <span className="hidden max-w-36 truncate text-sm font-medium md:inline">{name}</span>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-64">
        <DropdownMenuGroup>
          <DropdownMenuLabel className="space-y-1">
            <p className="truncate font-semibold">{name}</p>
            <p className="truncate text-xs font-normal text-muted-foreground" dir="ltr">
              {user.email}
            </p>
            <Badge variant="secondary" className="mt-1">
              {roles.has(user.role) ? roles(user.role) : user.role}
            </Badge>
          </DropdownMenuLabel>
        </DropdownMenuGroup>
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={handleLogout} className="text-destructive">
          <LogOut className="size-4" aria-hidden />
          {t("logout")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

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
    <header className="flex h-14 items-center gap-3 border-b bg-background px-4 md:px-6 print:hidden">
      <SidebarTrigger />
      <span className="truncate text-sm font-medium text-muted-foreground">{t("appName")}</span>
      <div className="flex-1" />
      <NotificationsBell />
      <Button variant="ghost" size="sm" onClick={toggleLocale}>
        <Languages className="size-4" />
        <span className="ms-2 hidden text-xs sm:inline">
          {locale === "ar" ? "English" : "العربية"}
        </span>
      </Button>
      <UserMenu />
    </header>
  );
}
