import { useTranslations } from "next-intl";
import { Link } from "@/i18n/routing";
import { UniversityLogo } from "@/components/brand/university-logo";

/** Public footer on university navy with the light logo. */
export function SiteFooter() {
  const t = useTranslations("site");
  const year = new Date().getFullYear();

  return (
    <footer className="mt-16 bg-sidebar text-sidebar-foreground">
      <div className="mx-auto flex max-w-6xl flex-col items-center gap-6 px-4 py-10 md:flex-row md:justify-between">
        <UniversityLogo variant="onDark" height={64} alt={t("logoAlt")} />
        <nav aria-label={t("nav.label")} className="flex flex-wrap justify-center gap-x-6 gap-y-2 text-sm">
          <Link href="/" className="hover:underline focus-visible:underline">{t("nav.home")}</Link>
          <Link href="/news" className="hover:underline focus-visible:underline">{t("nav.news")}</Link>
          <Link href="/portal" className="hover:underline focus-visible:underline">{t("footer.portal")}</Link>
        </nav>
      </div>
      <div className="border-t border-sidebar-border">
        <p className="mx-auto max-w-6xl px-4 py-4 text-center text-xs text-sidebar-foreground/80">
          {t("footer.rights", { year })}
        </p>
      </div>
    </footer>
  );
}
