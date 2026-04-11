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
  const t = useTranslations("nav");

  const mainItems: NavItem[] = [
    { href: "/dashboard", label: t("dashboard"), icon: LayoutDashboard },
    { href: "/researchers", label: t("researchers"), icon: Users },
    { href: "/publications", label: t("publications"), icon: FileText },
    { href: "/departments", label: t("departments"), icon: Building2 },
    { href: "/colleges", label: t("colleges"), icon: GraduationCap },
  ];

  const insightItems: NavItem[] = [
    { href: "/insights/trends", label: t("trends"), icon: TrendingUp },
    { href: "/search", label: t("search"), icon: Search },
  ];

  const adminItems: NavItem[] = [
    { href: "/admin/sync", label: t("sync"), icon: RefreshCw },
    { href: "/admin/settings", label: t("settings"), icon: Settings },
  ];

  return (
    <Sidebar>
      <SidebarHeader className="border-b px-6 py-4">
        <h1 className="text-lg font-bold">تراث</h1>
        <p className="text-xs text-muted-foreground">Research Performance</p>
      </SidebarHeader>
      <SidebarContent>
        <NavGroup title="الرئيسية" items={mainItems} />
        <NavGroup title="التحليلات" items={insightItems} />
        <NavGroup title="الإدارة" items={adminItems} />
      </SidebarContent>
    </Sidebar>
  );
}
