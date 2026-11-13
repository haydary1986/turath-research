"use client";

import { useLocale, useTranslations } from "next-intl";
import {
  LayoutDashboard,
  Users,
  FileText,
  GraduationCap,
  TrendingUp,
  Search,
  ClipboardList,
  FilePlus2,
  Mail,
  UserCog,
  FileUp,
  CalendarDays,
  SearchCheck,
  ShieldX,
  ShieldAlert,
  FileSpreadsheet,
  Inbox,
  BookUser,
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

const REVIEW_QUEUE_HREF = "/portal/requests?queue=true";
const REVIEWING_ROLES = ["admin", "college_dean", "scientific_affairs", "vp_research"];

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
                isActive={pathname === item.href.split("?")[0] && !item.href.includes("?")}
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
  const locale = useLocale();
  const nav = useTranslations("nav");
  const sb = useTranslations("sidebar");
  const pt = useTranslations("portal.nav");
  const role = useAuth((state) => state.user?.role);
  // Role comes from localStorage, so wait for hydration to keep server and client markup identical.
  const mounted = useHydrated();

  const reviewHref = mounted && role && REVIEWING_ROLES.includes(role) ? REVIEW_QUEUE_HREF : undefined;
  const staff = mounted && isStaffRole(role);

  const portalItems: NavItem[] = [
    { href: "/portal", label: pt("dashboard"), icon: ClipboardList },
    { href: "/portal/requests", label: pt("requests"), icon: FilePlus2 },
    ...(reviewHref ? [{ href: reviewHref, label: pt("reviewQueue"), icon: Inbox }] : []),
    { href: "/portal/profile", label: pt("profile"), icon: BookUser },
    { href: "/portal/activities", label: pt("activities"), icon: CalendarDays },
    { href: "/portal/journal-check", label: pt("journalCheck"), icon: SearchCheck },
    { href: "/portal/predatory-journals", label: pt("predatory"), icon: ShieldX },
    ...(staff
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

  // Legacy research-tracking screens: staff-only; researchers and deans work in the portal.
  const researchItems: NavItem[] = [
    { href: "/dashboard", label: nav("dashboard"), icon: LayoutDashboard },
    { href: "/researchers", label: nav("researchers"), icon: Users },
    { href: "/publications", label: nav("publications"), icon: FileText },
    { href: "/insights/trends", label: nav("trends"), icon: TrendingUp },
    { href: "/search", label: nav("search"), icon: Search },
  ];

  return (
    <Sidebar side={locale === "ar" ? "right" : "left"}>
      <SidebarHeader className="border-b border-sidebar-border px-4 py-4">
        <div className="flex items-center gap-3">
          <div className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-sidebar-primary text-sidebar-primary-foreground">
            <GraduationCap className="size-5" aria-hidden />
          </div>
          <div className="min-w-0">
            <h1 className="truncate text-base font-bold">{sb("brand")}</h1>
            <p className="truncate text-xs text-sidebar-foreground/80">{sb("subtitle")}</p>
          </div>
        </div>
      </SidebarHeader>
      <SidebarContent>
        <NavGroup title={pt("group")} items={portalItems} />
        {staff && <NavGroup title={sb("main")} items={researchItems} />}
      </SidebarContent>
    </Sidebar>
  );
}
