"use client";

import { useLocale, useTranslations } from "next-intl";
import { Languages } from "lucide-react";
import { Link, usePathname } from "@/i18n/routing";
import { UniversityLogo } from "@/components/brand/university-logo";
import { cn } from "@/lib/utils";
import { PortalEntryButton } from "./portal-entry-button";

const NAV = [
  { href: "/", key: "home" },
  { href: "/news", key: "news" },
] as const;

function isActive(pathname: string, href: string): boolean {
  return href === "/" ? pathname === "/" : pathname === href || pathname.startsWith(`${href}/`);
}

/** Public header: university logo, navigation, language switch and the portal button. */
export function SiteHeader() {
  const t = useTranslations("site");
  const locale = useLocale();
  const pathname = usePathname();
  const otherLocale = locale === "ar" ? "en" : "ar";

  const links = NAV.map(({ href, key }) => (
    <Link
      key={href}
      href={href}
      aria-current={isActive(pathname, href) ? "page" : undefined}
      className={cn(
        "rounded-md px-3 py-2 text-sm font-medium whitespace-nowrap text-foreground/80 transition-colors hover:text-accent",
        "focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none",
        "aria-[current=page]:text-primary aria-[current=page]:shadow-[inset_0_-2px_0_var(--accent)]"
      )}
    >
      {t(`nav.${key}`)}
    </Link>
  ));

  return (
    <header className="sticky top-0 z-30 border-b bg-card/95 backdrop-blur supports-[backdrop-filter]:bg-card/85">
      <a
        href="#main"
        className="sr-only focus:not-sr-only focus:absolute focus:start-4 focus:top-2 focus:z-50 focus:rounded-md focus:bg-primary focus:px-3 focus:py-2 focus:text-primary-foreground"
      >
        {t("nav.skipToContent")}
      </a>
      <div className="mx-auto flex max-w-6xl items-center gap-3 px-4 py-2">
        <Link href="/" className="shrink-0 rounded-md focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none">
          <UniversityLogo variant="color" height={52} alt={t("logoAlt")} priority />
        </Link>
        <nav aria-label={t("nav.label")} className="ms-4 hidden items-center gap-1 md:flex">
          {links}
        </nav>
        <div className="ms-auto flex items-center gap-2">
          <Link
            href={pathname}
            locale={otherLocale}
            lang={otherLocale}
            className="inline-flex h-8 items-center gap-1 rounded-lg px-2 text-sm text-foreground/80 hover:bg-muted hover:text-foreground focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
          >
            <Languages className="size-4" aria-hidden />
            {t("nav.switchLanguage")}
          </Link>
          <PortalEntryButton />
        </div>
      </div>
      <nav aria-label={t("nav.label")} className="mx-auto flex max-w-6xl gap-1 overflow-x-auto px-2 pb-1 md:hidden">
        {links}
      </nav>
    </header>
  );
}
