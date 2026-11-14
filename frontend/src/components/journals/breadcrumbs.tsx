import { getTranslations } from "next-intl/server";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { Link } from "@/i18n/routing";
import { JsonLd } from "@/components/seo/json-ld";
import { breadcrumbLd, type Crumb } from "@/lib/seo/json-ld";
import { Bidi } from "@/components/i18n/bidi";

/** Visible breadcrumb trail plus its BreadcrumbList structured data. The last crumb is the current page. */
export async function Breadcrumbs({ locale, crumbs }: { locale: string; crumbs: Crumb[] }) {
  const t = await getTranslations({ locale, namespace: "journals.crumbs" });
  const Separator = locale === "ar" ? ChevronLeft : ChevronRight;
  const trail: Crumb[] = [{ name: t("home"), path: "/" }, ...crumbs];

  return (
    <>
      <nav aria-label={t("label")} className="text-sm text-muted-foreground">
        <ol className="flex flex-wrap items-center gap-1">
          {trail.map((crumb, i) => {
            const last = i === trail.length - 1;
            return (
              <li key={crumb.path} className="flex min-w-0 items-center gap-1">
                {last ? (
                  crumb.data ? (
                    <span aria-current="page" className="truncate font-medium text-foreground"><Bidi text={crumb.name} /></span>
                  ) : (
                    <span aria-current="page" className="truncate font-medium text-foreground">{crumb.label ?? crumb.name}</span>
                  )
                ) : (
                  <>
                    <Link href={crumb.path} className="hover:text-accent hover:underline">{crumb.name}</Link>
                    <Separator className="size-3.5 shrink-0" aria-hidden />
                  </>
                )}
              </li>
            );
          })}
        </ol>
      </nav>
      <JsonLd data={breadcrumbLd(locale, trail)} />
    </>
  );
}
