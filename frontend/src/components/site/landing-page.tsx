import { getTranslations } from "next-intl/server";
import { JsonLd } from "@/components/seo/json-ld";
import { faqLd } from "@/lib/seo/json-ld";
import type { LandingData } from "@/lib/site/public-data";
import { pickLocalized } from "@/lib/i18n/text-lang";
import { localizeFaq } from "@/lib/site/localize";
import { FaqList } from "./faq-list";
import { RichText } from "./rich-text";
import { DepartmentProfile } from "./landing/department-profile";
import { DirectorMessage } from "./landing/director-message";
import { Hero } from "./landing/hero";
import { HowToApply } from "./landing/how-to-apply";
import { JournalsPromo } from "./landing/journals-promo";
import { LatestAnnouncements } from "./landing/latest-announcements";
import { LatestPublications } from "./landing/latest-publications";
import { QuickLinks } from "./landing/quick-links";
import { Section } from "./landing/section";
import { ServicesGrid } from "./landing/services-grid";
import { StatsBar } from "./landing/stats-bar";

/**
 * The public landing page, rendered on the server from the site document and the
 * public aggregates. Every block hides itself when it has nothing to show, and a
 * backend outage leaves the hero, steps and journal links standing.
 */
export async function LandingPage({ locale, data }: { locale: string; data: LandingData }) {
  const t = await getTranslations({ locale, namespace: "site.home" });
  const tl = await getTranslations({ locale, namespace: "landing" });
  const { content } = data;
  const about = pickLocalized(content?.about, content?.about_en, locale);
  const faq = localizeFaq(content?.faq ?? [], locale);

  return (
    <>
      <Hero
        locale={locale}
        title={pickLocalized(content?.hero_title, content?.hero_title_en, locale)}
        subtitle={pickLocalized(content?.hero_subtitle, content?.hero_subtitle_en, locale)}
      />
      <StatsBar locale={locale} stats={data.stats} />

      {about && (
        <Section id="about" title={t("about")}>
          <RichText text={about.text} lang={about.lang} className="max-w-3xl text-base" />
        </Section>
      )}

      {content?.department && <DepartmentProfile locale={locale} department={content.department} />}

      <ServicesGrid locale={locale} title={t("services")} services={content?.services ?? []} />
      <QuickLinks locale={locale} links={content?.links ?? []} />
      <HowToApply locale={locale} />

      {content && !content.hide_publications && <LatestPublications locale={locale} items={data.publications} />}

      <LatestAnnouncements locale={locale} circulars={data.circulars} activities={data.activities} />
      <JournalsPromo locale={locale} />

      {content && <DirectorMessage locale={locale} director={content.director} />}

      {!!content?.faq.length && (
        <Section id="faq" title={tl("faq.title")}>
          <FaqList items={faq} />
          <JsonLd data={faqLd(faq)} />
        </Section>
      )}
    </>
  );
}
