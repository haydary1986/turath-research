import { getTranslations } from "next-intl/server";
import { ArrowLeft, ArrowRight, Lock } from "lucide-react";
import { BALANCED_GRID, balancedCardClass } from "@/lib/site/grid";
import { isSafeServiceLink, requiresSignIn, type SiteService } from "@/lib/site/types";
import { cn } from "@/lib/utils";
import { Localized } from "@/components/i18n/localized";
import { ServiceIcon } from "../service-icon";
import { Section } from "./section";
import { ServiceLink } from "./service-link";

interface ServicesGridProps {
  locale: string;
  title: string;
  services: SiteService[];
}

function ServiceBody({ service, locale, cta, locked }: { service: SiteService; locale: string; cta?: string; locked?: string }) {
  const Forward = locale === "ar" ? ArrowLeft : ArrowRight;
  return (
    <>
      <span className="flex size-12 items-center justify-center rounded-xl bg-primary text-primary-foreground transition-colors group-hover:bg-accent">
        <ServiceIcon name={service.icon} className="size-6" />
      </span>
      <Localized as="h3" block ar={service.title} en={service.title_en} locale={locale} className="mt-4 text-lg font-bold text-primary" />
      {/* Plain text: the card itself is a link, so the description must not contain links. */}
      <Localized as="p" block ar={service.description} en={service.description_en} locale={locale}
        className="mt-2 text-sm leading-relaxed whitespace-pre-line text-muted-foreground" />
      {cta && (
        <span className="mt-auto inline-flex items-center gap-1.5 pt-4 text-sm font-semibold text-accent">
          {cta}
          <Forward className="size-4 transition-transform rtl:group-hover:-translate-x-0.5 ltr:group-hover:translate-x-0.5" aria-hidden />
          {locked && (
            <span className="ms-auto inline-flex items-center gap-1 text-xs font-normal text-muted-foreground">
              <Lock className="size-3" aria-hidden />
              {locked}
            </span>
          )}
        </span>
      )}
    </>
  );
}

/** Service cards with icons; each links to its service when a safe link is set. */
export async function ServicesGrid({ locale, title, services }: ServicesGridProps) {
  if (services.length === 0) return null;
  const t = await getTranslations({ locale, namespace: "landing.services" });
  const card =
    "group flex h-full flex-col rounded-2xl border bg-card p-5 transition-shadow focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none";

  return (
    <Section id="services" title={title}>
      <ul className={BALANCED_GRID}>
        {services.map((service, i) => {
          const link = isSafeServiceLink(service.link) ? service.link : "";
          return (
            <li key={i} className={balancedCardClass(i, services.length)}>
              {link ? (
                <ServiceLink href={link} className={cn(card, "hover:border-accent/40 hover:shadow-md")}>
                  <ServiceBody service={service} locale={locale} cta={t("open")}
                    locked={requiresSignIn(link) ? t("signInFirst") : undefined} />
                </ServiceLink>
              ) : (
                <div className={card}>
                  <ServiceBody service={service} locale={locale} />
                </div>
              )}
            </li>
          );
        })}
      </ul>
    </Section>
  );
}
