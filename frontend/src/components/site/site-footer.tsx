import type { ReactNode } from "react";
import { getTranslations } from "next-intl/server";
import { Clock, ExternalLink, Mail, MapPin, Phone, type LucideIcon } from "lucide-react";
import { Link } from "@/i18n/routing";
import { UniversityLogo } from "@/components/brand/university-logo";
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

/** Public footer on university navy: identity and contact, portal pages, research tools,
 * useful links, then the colleges across the full width and the rights line. */
export async function SiteFooter({ locale, content }: { locale: string; content: SiteContent | null }) {
  const t = await getTranslations({ locale, namespace: "site" });
  const tf = await getTranslations({ locale, namespace: "landing.footer" });
  const tn = await getTranslations({ locale, namespace: "landing.nav" });
  const year = new Date().getFullYear();
  const contact = content ? contactRows(content.contact, (key) => tf(key), locale) : [];
  const newTab = tf("opensInNewTab");

  return (
    <footer id="contact" className="mt-20 bg-sidebar text-sidebar-foreground">
      <div className="mx-auto grid max-w-6xl gap-10 px-4 py-12 sm:grid-cols-2 lg:grid-cols-[1.4fr_1fr_1.3fr_1.1fr]">
        <div className="space-y-5">
          <UniversityLogo variant="onDark" height={64} alt={t("logoAlt")} />
          <p className="max-w-xs text-sm leading-relaxed text-sidebar-foreground/80">{tf("tagline")}</p>
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
            <ul className="flex flex-wrap gap-2" aria-label={tf("followUs")}>
              {content.socials.map((social, i) => (
                <li key={i}>
                  <a
                    href={social.url}
                    target="_blank"
                    rel="noopener noreferrer me"
                    className="flex size-10 items-center justify-center rounded-full bg-white/10 text-white hover:bg-accent focus-visible:ring-2 focus-visible:ring-white/60 focus-visible:outline-none"
                  >
                    <SocialIcon platform={social.platform} className="size-5" />
                    <span className="sr-only">{tf(`social.${social.platform}`)} {newTab}</span>
                  </a>
                </li>
              ))}
            </ul>
          )}
        </div>

        <Column id="footer-portal" title={tf("portalTitle")}>
          <ul className="space-y-2 text-sm">
            <li><Link href="/" className={linkClass}>{t("nav.home")}</Link></li>
            <li><Link href="/news" className={linkClass}>{t("nav.news")}</Link></li>
            <li><Link href={JOURNAL_PATHS.guide} className={linkClass}>{tn("journalsGuide")}</Link></li>
            <li><Link href="/portal" className={linkClass}>{t("footer.portal")}</Link></li>
          </ul>
        </Column>

        {!!content?.tools.length && (
          <Column id="footer-tools" title={tf("toolsTitle")}>
            <ExternalLinks links={content.tools} newTab={newTab} locale={locale} />
          </Column>
        )}

        {!!content?.links.length && (
          <Column id="footer-links" title={tf("linksTitle")}>
            <ExternalLinks links={content.links} newTab={newTab} locale={locale} />
          </Column>
        )}
      </div>

      {!!content?.colleges.length && (
        <div className="border-t border-sidebar-border">
          <section aria-labelledby="footer-colleges" className="mx-auto max-w-6xl px-4 py-8">
            <h2 id="footer-colleges" className="mb-4 text-sm font-bold tracking-wide text-white">{tf("collegesTitle")}</h2>
            <ul className="grid gap-x-6 gap-y-2 text-sm sm:grid-cols-2 lg:grid-cols-4">
              {content.colleges.map((college, i) => (
                <li key={i}>
                  <a href={college.url} target="_blank" rel="noopener noreferrer" className={`${linkClass} inline-flex items-center gap-1.5`}>
                    <Localized ar={college.label} en={college.label_en} locale={locale} />
                    <ExternalLink className="size-3 opacity-70" aria-hidden />
                    <span className="sr-only">{newTab}</span>
                  </a>
                </li>
              ))}
            </ul>
          </section>
        </div>
      )}

      <div className="border-t border-sidebar-border">
        <p className="mx-auto max-w-6xl px-4 py-4 text-center text-xs text-sidebar-foreground/80">
          {t("footer.rights", { year })}
        </p>
      </div>
    </footer>
  );
}
