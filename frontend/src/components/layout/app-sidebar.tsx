"use client";

import { useEffect, useState } from "react";
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
  ClipboardCheck,
  ShieldCheck,
} from "lucide-react";
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuBadge,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";
import { Link, usePathname } from "@/i18n/routing";
import { useHydrated } from "@/hooks/use-hydrated";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { isStaffRole } from "@/lib/portal/types";
import { canManageSite } from "@/lib/site/types";
import { useMyResponsibilities, useSessionRefresh } from "@/lib/portal/my-access";
import { UniversityLogo } from "@/components/brand/university-logo";
import type { LucideIcon } from "lucide-react";

interface NavItem {
  href: string;
  label: string;
  icon: LucideIcon;
  badge?: number;
}

interface NavSection {
  key: string;
  title?: string;
  items: NavItem[];
}

const REVIEW_QUEUE_HREF = "/portal/requests?queue=true";
const REVIEWING_ROLES = ["admin", "college_dean", "scientific_affairs", "vp_research"];
const QUEUE_REFRESH_MS = 5 * 60 * 1000;

/** A link is active on its own page and its sub-pages; query links (the review queue) never claim it. */
function isActive(pathname: string, href: string): boolean {
  if (href.includes("?")) return false;
  if (href === "/portal") return pathname === "/portal";
  return pathname === href || pathname.startsWith(`${href}/`);
}

/** Number of requests waiting for the signed-in reviewer, refreshed periodically and on focus. */
function useQueueCount(enabled: boolean): number {
  const [count, setCount] = useState(0);

  useEffect(() => {
    if (!enabled) return;
    let active = true;
    const load = () =>
      api
        .get<unknown[]>("/api/v1/portal/requests?queue=true&limit=1")
        .then((res) => active && setCount(res.meta?.total ?? 0))
        .catch(() => undefined);
    load();
    const timer = setInterval(load, QUEUE_REFRESH_MS);
    window.addEventListener("focus", load);
    return () => {
      active = false;
      clearInterval(timer);
      window.removeEventListener("focus", load);
    };
  }, [enabled]);

  return enabled ? count : 0;
}

function NavGroup({ title, items }: { title?: string; items: NavItem[] }) {
  const pathname = usePathname();
  if (items.length === 0) return null;

  return (
    <SidebarGroup className="py-1">
      {title && <SidebarGroupLabel>{title}</SidebarGroupLabel>}
      <SidebarGroupContent>
        <SidebarMenu>
          {items.map((item) => (
            <SidebarMenuItem key={item.href}>
              <SidebarMenuButton
                className="relative data-active:before:absolute data-active:before:inset-y-1.5 data-active:before:start-0 data-active:before:w-1 data-active:before:rounded-full data-active:before:bg-sidebar-primary"
                isActive={isActive(pathname, item.href)}
                render={<Link href={item.href} />}
              >
                <item.icon className="size-4" />
                <span>{item.label}</span>
              </SidebarMenuButton>
              {item.badge ? (
                <SidebarMenuBadge className="rounded-full bg-sidebar-primary text-sidebar-primary-foreground">
                  {item.badge > 99 ? "99+" : item.badge}
                </SidebarMenuBadge>
              ) : null}
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
  const ui = useTranslations("portal.ui");
  const role = useAuth((state) => state.user?.role);
  // Role comes from localStorage, so wait for hydration to keep server and client markup identical.
  const mounted = useHydrated();
  // Re-reads the user (role, college) and responsibilities: on mount, every few minutes, on focus.
  useSessionRefresh();
  const holdsResponsibilities = useMyResponsibilities().length > 0;

  const reviewer = mounted && !!role && REVIEWING_ROLES.includes(role);
  const staff = mounted && isStaffRole(role);
  const admin = mounted && role === "admin";
  const siteManager = mounted && canManageSite(role);
  const queueCount = useQueueCount(reviewer);

  const sections: NavSection[] = [
    {
      key: "home",
      items: [{ href: "/portal", label: pt("dashboard"), icon: ClipboardList }],
    },
    {
      key: "review",
      title: pt("groupReview"),
      items: reviewer
        ? [
            { href: REVIEW_QUEUE_HREF, label: pt("reviewQueue"), icon: Inbox, badge: queueCount },
            ...(staff ? [{ href: "/portal/requests", label: pt("allRequests"), icon: FilePlus2 }] : []),
          ]
        : [],
    },
    {
      key: "work",
      title: pt("groupWork"),
      items: [
        ...(staff ? [] : [{ href: "/portal/requests", label: ui("myRequests"), icon: FilePlus2 }]),
        { href: "/portal/activities", label: pt("activities"), icon: CalendarDays },
        { href: "/portal/profile", label: pt("profile"), icon: BookUser },
        ...(mounted && holdsResponsibilities
          ? [{ href: "/portal/responsibilities", label: pt("myResponsibilities"), icon: ClipboardCheck }]
          : []),
      ],
    },
    {
      key: "tools",
      title: pt("groupTools"),
      items: [
        { href: "/portal/journal-check", label: pt("journalCheck"), icon: SearchCheck },
        { href: "/portal/predatory-journals", label: pt("predatory"), icon: ShieldX },
        ...(staff ? [{ href: "/portal/journals", label: pt("journals"), icon: ShieldAlert }] : []),
      ],
    },
    {
      key: "research",
      title: pt("groupResearch"),
      // Research-tracking screens over the whole staff register: staff only.
      items: staff
        ? [
            { href: "/dashboard", label: nav("dashboard"), icon: LayoutDashboard },
            { href: "/researchers", label: nav("researchers"), icon: Users },
            { href: "/publications", label: nav("publications"), icon: FileText },
            { href: "/insights/trends", label: nav("trends"), icon: TrendingUp },
            { href: "/search", label: nav("search"), icon: Search },
          ]
        : [],
    },
    {
      key: "outputs",
      title: pt("groupOutputs"),
      items: staff
        ? [
            { href: "/portal/templates", label: pt("templates"), icon: Mail },
            { href: "/portal/reports", label: pt("reports"), icon: FileSpreadsheet },
          ]
        : [],
    },
    {
      key: "admin",
      title: pt("groupAdmin"),
      items: [
        ...(siteManager ? [{ href: "/portal/site", label: pt("site"), icon: Globe }] : []),
        ...(admin
          ? [
              { href: "/portal/users", label: pt("users"), icon: UserCog },
              { href: "/portal/import", label: pt("import"), icon: FileUp },
              { href: "/portal/audit", label: pt("audit"), icon: ShieldCheck },
            ]
          : []),
      ],
    },
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
      <SidebarContent className="gap-0 pb-4">
        {sections.map((section) => (
          <NavGroup key={section.key} title={section.title} items={section.items} />
        ))}
      </SidebarContent>
    </Sidebar>
  );
}
