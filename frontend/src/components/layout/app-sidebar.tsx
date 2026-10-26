"use client";

import { useTranslations } from "next-intl";
import {
  LayoutDashboard,
  Users,
  FileText,
  Building2,
  GraduationCap,
  TrendingUp,
  Search,
  Settings,
  RefreshCw,
  ClipboardList,
  FilePlus2,
  Mail,
  UserCog,
  FileUp,
  CalendarDays,
  ShieldAlert,
  FileSpreadsheet,
} from "lucide-react";
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";
import { Link, usePathname } from "@/i18n/routing";
import { useHydrated } from "@/hooks/use-hydrated";
import { useAuth } from "@/lib/auth";
import { isStaffRole } from "@/lib/portal/types";
import type { LucideIcon } from "lucide-react";

interface NavItem {
  href: string;
  label: string;
  icon: LucideIcon;
}

function NavGroup({ title, items }: { title: string; items: NavItem[] }) {
  const pathname = usePathname();

  return (
    <SidebarGroup>
      <SidebarGroupLabel>{title}</SidebarGroupLabel>
      <SidebarGroupContent>
        <SidebarMenu>
          {items.map((item) => (
            <SidebarMenuItem key={item.href}>
              <SidebarMenuButton
                isActive={pathname === item.href}
                render={<Link href={item.href} />}
              >
                <item.icon className="size-4" />
                <span>{item.label}</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          ))}
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  );
}

export function AppSidebar() {
  const nav = useTranslations("nav");
  const sb = useTranslations("sidebar");
  const pt = useTranslations("portal.nav");
  const role = useAuth((state) => state.user?.role);
  // Role comes from localStorage, so wait for hydration to keep server and client markup identical.
  const mounted = useHydrated();

  const portalItems: NavItem[] = [
    { href: "/portal", label: pt("dashboard"), icon: ClipboardList },
    { href: "/portal/requests", label: pt("requests"), icon: FilePlus2 },
    { href: "/portal/activities", label: pt("activities"), icon: CalendarDays },
    ...(mounted && isStaffRole(role)
      ? [
          { href: "/portal/templates", label: pt("templates"), icon: Mail },
          { href: "/portal/journals", label: pt("journals"), icon: ShieldAlert },
          { href: "/portal/reports", label: pt("reports"), icon: FileSpreadsheet },
        ]
      : []),
    ...(mounted && role === "admin"
      ? [
          { href: "/portal/users", label: pt("users"), icon: UserCog },
          { href: "/portal/import", label: pt("import"), icon: FileUp },
        ]
      : []),
  ];

  const mainItems: NavItem[] = [
    { href: "/dashboard", label: nav("dashboard"), icon: LayoutDashboard },
    { href: "/researchers", label: nav("researchers"), icon: Users },
    { href: "/publications", label: nav("publications"), icon: FileText },
    { href: "/departments", label: nav("departments"), icon: Building2 },
    { href: "/colleges", label: nav("colleges"), icon: GraduationCap },
  ];

  const insightItems: NavItem[] = [
    { href: "/insights/trends", label: nav("trends"), icon: TrendingUp },
    { href: "/search", label: nav("search"), icon: Search },
  ];

  const adminItems: NavItem[] = [
    { href: "/admin/sync", label: nav("sync"), icon: RefreshCw },
    { href: "/admin/settings", label: nav("settings"), icon: Settings },
  ];

  return (
    <Sidebar>
      <SidebarHeader className="border-b border-sidebar-border px-4 py-4">
        <div className="flex items-center gap-3">
          <div className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-sidebar-primary text-sidebar-primary-foreground">
            <GraduationCap className="size-5" aria-hidden />
          </div>
          <div className="min-w-0">
            <h1 className="truncate text-base font-bold">{sb("brand")}</h1>
            <p className="truncate text-xs opacity-70">{sb("subtitle")}</p>
          </div>
        </div>
      </SidebarHeader>
      <SidebarContent>
        <NavGroup title={pt("group")} items={portalItems} />
        <NavGroup title={sb("main")} items={mainItems} />
        <NavGroup title={sb("insights")} items={insightItems} />
        <NavGroup title={sb("admin")} items={adminItems} />
      </SidebarContent>
    </Sidebar>
  );
}
