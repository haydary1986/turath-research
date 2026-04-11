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
      <SidebarHeader className="border-b px-6 py-4">
        <h1 className="text-lg font-bold">{sb("brand")}</h1>
        <p className="text-xs text-muted-foreground">{sb("subtitle")}</p>
      </SidebarHeader>
      <SidebarContent>
        <NavGroup title={sb("main")} items={mainItems} />
        <NavGroup title={sb("insights")} items={insightItems} />
        <NavGroup title={sb("admin")} items={adminItems} />
      </SidebarContent>
    </Sidebar>
  );
}
