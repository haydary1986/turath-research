import type { ReactNode } from "react";
import { getTranslations } from "next-intl/server";
import { Clock, ExternalLink, Mail, MapPin, Phone, type LucideIcon } from "lucide-react";
import { Link } from "@/i18n/routing";
import { UniversityLogo } from "@/components/brand/university-logo";
import { articlesFor, TOPIC_SLUGS } from "@/lib/journals/articles";
import { JOURNAL_PATHS } from "@/lib/journals/lists";
import type { SiteContent, SiteLink } from "@/lib/site/types";
import { Localized } from "@/components/i18n/localized";
import { SocialIcon } from "./social-icon";

const linkClass =
  "rounded-sm text-sidebar-foreground/85 hover:text-white hover:underline focus-visible:ring-2 focus-visible:ring-white/60 focus-visible:outline-none";

function Column({ id, title, children }: { id: string; title: string; children: ReactNode }) {
  return (
    <section aria-labelledby={id} className="min-w-0">
      <h2 id={id} className="mb-3 text-sm font-bold tracking-wide text-white">{title}</h2>
      {children}
    </section>
  );
}

function ExternalLinks({ links, newTab, locale }: { links: SiteLink[]; newTab: string; locale: string }) {
  return (
    <ul className="space-y-2 text-sm">
      {links.map((link, i) => (
        <li key={i}>
          <a href={link.url} target="_blank" rel="noopener noreferrer" className={`${linkClass} inline-flex items-center gap-1.5`}>
            <Localized ar={link.label} en={link.label_en} locale={locale} />
            <ExternalLink className="size-3 opacity-70" aria-hidden />
            <span className="sr-only">{newTab}</span>
          </a>
        </li>
      ))}
    </ul>
  );
}

interface ContactRow {
  icon: LucideIcon;
  label: string;
  value: ReactNode;
}

function contactRows(contact: SiteContent["contact"], t: (key: string) => string, locale: string): ContactRow[] {
  const rows: (ContactRow | false)[] = [
    !!contact.email && {
      icon: Mail, label: t("email"),
      value: <a href={`mailto:${contact.email}`} dir="ltr" data-bidi="" className={linkClass}>{contact.email}</a>,
    },
    !!contact.phone && {
      icon: Phone, label: t("phone"),
      value: <a href={`tel:${contact.phone.replace(/[^\d+]/g, "")}`} dir="ltr" data-bidi="" className={linkClass}>{contact.phone}</a>,
    },
    !!(contact.location || contact.location_en) && {
      icon: MapPin, label: t("location"), value: <Localized ar={contact.location} en={contact.location_en} locale={locale} />,
    },
    !!(contact.hours || contact.hours_en) && {
      icon: Clock, label: t("hours"), value: <Localized ar={contact.hours} en={contact.hours_en} locale={locale} />,
    },
  ];
  return rows.filter((row): row is ContactRow => !!row);
}

/** Public footer on university navy: contact, journal guide, colleges, links and social accounts. */
export async function SiteFooter({ locale, content }: { locale: string; content: SiteContent | null }) {
  const t = await getTranslations({ locale, namespace: "site" });
  const tf = await getTranslations({ locale, namespace: "landing.footer" });
  const tn = await getTranslations({ locale, namespace: "landing.nav" });
  const articles = articlesFor(locale);
  const year = new Date().getFullYear();
  const contact = content ? contactRows(content.contact, (key) => tf(key), locale) : [];

  return (
    <footer id="contact" className="mt-20 bg-sidebar text-sidebar-foreground">
      <div className="mx-auto grid max-w-6xl gap-10 px-4 py-12 sm:grid-cols-2 lg:grid-cols-[1.3fr_1fr_1.2fr_1fr]">
        <div className="space-y-5">
          <UniversityLogo variant="onDark" height={64} alt={t("logoAlt")} />
          {contact.length > 0 && (
            <Column id="footer-contact" title={tf("contactTitle")}>
              <dl className="space-y-2.5 text-sm">
                {contact.map(({ icon: Icon, label, value }) => (
                  <div key={label} className="flex items-start gap-2.5">
                    <Icon className="mt-0.5 size-4 shrink-0 text-white/70" aria-hidden />
                    <div className="min-w-0">
                      <dt className="sr-only">{label}</dt>
                      <dd className="break-words">{value}</dd>
                    </div>
                  </div>
                ))}
              </dl>
            </Column>
          )}
          {!!content?.socials.length && (
            <Column id="footer-social" title={tf("followUs")}>
              <ul className="flex flex-wrap gap-2">
                {content.socials.map((social, i) => (
                  <li key={i}>
                    <a
                      href={social.url}
                      target="_blank"
                      rel="noopener noreferrer me"
                      className="flex size-10 items-center justify-center rounded-full bg-white/10 text-white hover:bg-accent focus-visible:ring-2 focus-visible:ring-white/60 focus-visible:outline-none"
                    >
                      <SocialIcon platform={social.platform} className="size-5" />
                      <span className="sr-only">{tf(`social.${social.platform}`)} {tf("opensInNewTab")}</span>
                    </a>
                  </li>
                ))}
              </ul>
            </Column>
          )}
        </div>

        <Column id="footer-journals" title={tf("journalsTitle")}>
          <ul className="space-y-2 text-sm">
            <li><Link href={JOURNAL_PATHS.guide} className={linkClass}>{tn("journalsGuide")}</Link></li>
            <li><Link href={JOURNAL_PATHS.check} className={linkClass}>{tn("journalCheck")}</Link></li>
            {TOPIC_SLUGS.map((slug) => (
              <li key={slug}><Link href={JOURNAL_PATHS.topic(slug)} className={linkClass}>{articles[slug].navLabel}</Link></li>
            ))}
          </ul>
        </Column>

        {!!content?.colleges.length && (
          <Column id="footer-colleges" title={tf("collegesTitle")}>
            <ExternalLinks links={content.colleges} newTab={tf("opensInNewTab")} locale={locale} />
          </Column>
        )}

        <Column id="footer-links" title={tf("linksTitle")}>
          <ul className="mb-2 space-y-2 text-sm">
            <li><Link href="/" className={linkClass}>{t("nav.home")}</Link></li>
            <li><Link href="/news" className={linkClass}>{t("nav.news")}</Link></li>
            <li><Link href="/portal" className={linkClass}>{t("footer.portal")}</Link></li>
          </ul>
          {!!content?.links.length && <ExternalLinks links={content.links} newTab={tf("opensInNewTab")} locale={locale} />}
        </Column>
      </div>
      <div className="border-t border-sidebar-border">
        <p className="mx-auto max-w-6xl px-4 py-4 text-center text-xs text-sidebar-foreground/80">
          {t("footer.rights", { year })}
        </p>
      </div>
    </footer>
  );
}
