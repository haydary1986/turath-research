"use client";

import type { ReactNode } from "react";
import { useLocale, useTranslations } from "next-intl";
import { ArrowLeft, ArrowRight, Clock, ExternalLink, Mail, MapPin, Phone } from "lucide-react";
import { Link } from "@/i18n/routing";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { UniversityLogo } from "@/components/brand/university-logo";
import { PUBLIC_API, type AnnouncementType, type PublicAnnouncement, type SiteContent } from "@/lib/site/types";
import { AnnouncementCard } from "./announcement-card";
import { PortalEntryButton } from "./portal-entry-button";
import { RichText } from "./rich-text";
import { usePublicResource } from "./use-public-resource";

const LATEST_COUNT = 4;

function Section({ id, title, children, action }: { id: string; title: string; children: ReactNode; action?: ReactNode }) {
  return (
    <section aria-labelledby={`${id}-title`} className="mx-auto max-w-6xl px-4 pt-12">
      <div className="mb-5 flex items-end justify-between gap-4">
        <h2 id={`${id}-title`} className="border-s-4 border-accent ps-3 text-xl font-bold text-primary md:text-2xl">
          {title}
        </h2>
        {action}
      </div>
      {children}
    </section>
  );
}

function Hero({ content }: { content: SiteContent | null }) {
  const t = useTranslations("site");
  return (
    <section className="bg-gradient-to-l from-sidebar to-[oklch(0.239_0.101_263.3)] text-sidebar-foreground">
      <div className="mx-auto grid max-w-6xl items-center gap-8 px-4 py-12 md:grid-cols-[1fr_auto] md:py-16">
        <div className="min-w-0">
          {content ? (
            <>
              <h1 className="text-3xl leading-tight font-bold md:text-4xl">{content.hero_title}</h1>
              {content.hero_subtitle && (
                <p className="mt-4 max-w-2xl text-base leading-relaxed text-sidebar-foreground/90 md:text-lg">
                  {content.hero_subtitle}
                </p>
              )}
            </>
          ) : (
            <div className="space-y-3" aria-hidden>
              <Skeleton className="h-10 w-2/3 bg-white/15" />
              <Skeleton className="h-5 w-full max-w-xl bg-white/10" />
            </div>
          )}
          <div className="mt-8 flex flex-wrap gap-3">
            <PortalEntryButton size="lg" tone="onDark" />
            <Button
              size="lg"
              variant="outline"
              className="h-11 border-white/40 bg-transparent px-5 text-base text-white hover:bg-white/10 hover:text-white"
              render={<Link href="/news" />}
            >
              {t("nav.news")}
            </Button>
          </div>
          <p className="mt-3 text-sm text-sidebar-foreground/80">{t("nav.signInHint")}</p>
        </div>
        <div className="hidden md:block">
          <UniversityLogo variant="crest" height={220} alt={t("logoAlt")} priority />
        </div>
      </div>
    </section>
  );
}

function ContactList({ contact }: { contact: SiteContent["contact"] }) {
  const t = useTranslations("site.home");
  const rows = [
    contact.email && { icon: Mail, label: t("email"), value: <a href={`mailto:${contact.email}`} dir="ltr" className="text-primary hover:text-accent hover:underline">{contact.email}</a> },
    contact.phone && { icon: Phone, label: t("phone"), value: <a href={`tel:${contact.phone.replace(/[^\d+]/g, "")}`} dir="ltr" className="text-primary hover:text-accent hover:underline">{contact.phone}</a> },
    contact.location && { icon: MapPin, label: t("location"), value: contact.location },
    contact.hours && { icon: Clock, label: t("hours"), value: contact.hours },
  ].filter(Boolean) as { icon: typeof Mail; label: string; value: ReactNode }[];
  if (rows.length === 0) return null;
  return (
    <dl className="grid gap-4 sm:grid-cols-2">
      {rows.map(({ icon: Icon, label, value }) => (
        <div key={label} className="flex items-start gap-3 rounded-xl border bg-card p-4">
          <Icon className="mt-0.5 size-5 shrink-0 text-accent" aria-hidden />
          <div className="min-w-0">
            <dt className="text-xs text-muted-foreground">{label}</dt>
            <dd className="mt-0.5 font-medium break-words">{value}</dd>
          </div>
        </div>
      ))}
    </dl>
  );
}

function LatestList({ type }: { type: AnnouncementType }) {
  const t = useTranslations("site.home");
  const { data, error, loading, reload } = usePublicResource<PublicAnnouncement[]>(
    `${PUBLIC_API.announcements}?type=${type}&limit=${LATEST_COUNT}`
  );
  if (loading) {
    return (
      <div className="space-y-3" aria-hidden>
        {[0, 1, 2].map((i) => <Skeleton key={i} className="h-20 w-full" />)}
      </div>
    );
  }
  if (error) {
    return (
      <Alert variant="warning">
        {t("loadError")}{" "}
        <button type="button" onClick={reload} className="font-semibold underline">{t("retry")}</button>
      </Alert>
    );
  }
  if (!data?.length) {
    return <p className="rounded-xl border border-dashed p-6 text-center text-muted-foreground">{t(type === "circular" ? "noCirculars" : "noActivities")}</p>;
  }
  return (
    <div className="space-y-3">
      {data.map((item) => <AnnouncementCard key={item.id} item={item} compact />)}
    </div>
  );
}

/** The public landing page: hero, about, services, latest circulars/activities, quick links, contact. */
export function LandingPage() {
  const t = useTranslations("site");
  const locale = useLocale();
  const Forward = locale === "ar" ? ArrowLeft : ArrowRight;
  const { data: content, error, reload } = usePublicResource<SiteContent>(PUBLIC_API.site);
  const viewAll = (type: AnnouncementType) => (
    <Link href={`/news?type=${type}`} className="inline-flex items-center gap-1 text-sm font-semibold text-primary hover:text-accent">
      {t("home.viewAll")}
      <Forward className="size-4" aria-hidden />
    </Link>
  );

  return (
    <>
      <Hero content={content} />
      {error && (
        <div className="mx-auto max-w-6xl px-4 pt-6">
          <Alert variant="warning">
            {t("home.loadError")}{" "}
            <button type="button" onClick={reload} className="font-semibold underline">{t("home.retry")}</button>
          </Alert>
        </div>
      )}

      {content?.about && (
        <Section id="about" title={t("home.about")}>
          <RichText text={content.about} className="max-w-3xl text-base" />
        </Section>
      )}

      {!!content?.services.length && (
        <Section id="services" title={t("home.services")}>
          <ul className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {content.services.map((service, i) => (
              <li key={i} className="rounded-xl border border-t-4 border-t-primary bg-card p-5">
                <h3 className="font-bold text-primary">{service.title}</h3>
                {service.description && <RichText text={service.description} className="mt-2 text-sm text-muted-foreground" />}
              </li>
            ))}
          </ul>
        </Section>
      )}

      <div className="mx-auto grid max-w-6xl gap-0 lg:grid-cols-2 lg:gap-8">
        <Section id="circulars" title={t("home.latestCirculars")} action={viewAll("circular")}>
          <LatestList type="circular" />
        </Section>
        <Section id="activities" title={t("home.latestActivities")} action={viewAll("activity")}>
          <LatestList type="activity" />
        </Section>
      </div>

      {!!content?.links.length && (
        <Section id="links" title={t("home.quickLinks")}>
          <ul className="flex flex-wrap gap-3">
            {content.links.map((link, i) => (
              <li key={i}>
                <a
                  href={link.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-2 rounded-lg border bg-card px-4 py-2 text-sm font-medium text-primary hover:border-accent hover:text-accent focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
                >
                  {link.label}
                  <ExternalLink className="size-3.5" aria-hidden />
                  <span className="sr-only">{t("home.opensInNewTab")}</span>
                </a>
              </li>
            ))}
          </ul>
        </Section>
      )}

      {content && (content.contact.email || content.contact.phone || content.contact.location || content.contact.hours) && (
        <Section id="contact" title={t("home.contact")}>
          <ContactList contact={content.contact} />
        </Section>
      )}
    </>
  );
}
