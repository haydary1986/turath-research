"use client";

import { useLocale, useTranslations } from "next-intl";
import {
  LayoutDashboard,
  Users,
  FileText,
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
  Globe,
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
import { canManageSite } from "@/lib/site/types";
import { UniversityLogo } from "@/components/brand/university-logo";
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
                className="relative data-active:before:absolute data-active:before:inset-y-1.5 data-active:before:start-0 data-active:before:w-1 data-active:before:rounded-full data-active:before:bg-sidebar-primary"
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
  const site = useTranslations("site");
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
    ...(mounted && canManageSite(role) ? [{ href: "/portal/site", label: pt("site"), icon: Globe }] : []),
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
        <Link href="/" className="flex items-center gap-3 rounded-md outline-none focus-visible:ring-2 focus-visible:ring-sidebar-ring">
          <UniversityLogo variant="crest" height={44} alt={site("logoAlt")} priority />
          <div className="min-w-0">
            <p className="truncate text-base font-bold">{sb("brand")}</p>
            <p className="truncate text-xs text-sidebar-foreground/80">{sb("subtitle")}</p>
          </div>
        </Link>
      </SidebarHeader>
      <SidebarContent>
        <NavGroup title={pt("group")} items={portalItems} />
        {staff && <NavGroup title={sb("main")} items={researchItems} />}
      </SidebarContent>
    </Sidebar>
  );
}
