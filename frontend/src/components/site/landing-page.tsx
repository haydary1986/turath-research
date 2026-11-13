import { getTranslations } from "next-intl/server";
import { JsonLd } from "@/components/seo/json-ld";
import { faqLd } from "@/lib/seo/json-ld";
import type { LandingData } from "@/lib/site/public-data";
import { FaqList } from "./faq-list";
import { RichText } from "./rich-text";
import { DirectorMessage } from "./landing/director-message";
import { Hero } from "./landing/hero";
import { HowToApply } from "./landing/how-to-apply";
import { JournalsPromo } from "./landing/journals-promo";
import { LatestAnnouncements } from "./landing/latest-announcements";
import { LatestPublications } from "./landing/latest-publications";
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

  return (
    <>
      <Hero locale={locale} title={content?.hero_title ?? null} subtitle={content?.hero_subtitle ?? null} />
      <StatsBar locale={locale} stats={data.stats} />

      {content?.about && (
        <Section id="about" title={t("about")}>
          <RichText text={content.about} className="max-w-3xl text-base" />
        </Section>
      )}

      <ServicesGrid locale={locale} title={t("services")} services={content?.services ?? []} />
      <HowToApply locale={locale} />

      {content && !content.hide_publications && <LatestPublications locale={locale} items={data.publications} />}

      <LatestAnnouncements locale={locale} circulars={data.circulars} activities={data.activities} />
      <JournalsPromo locale={locale} />

      {content && <DirectorMessage locale={locale} director={content.director} />}

      {!!content?.faq.length && (
        <Section id="faq" title={tl("faq.title")}>
          <FaqList items={content.faq} />
          <JsonLd data={faqLd(content.faq)} />
        </Section>
      )}
    </>
  );
}
