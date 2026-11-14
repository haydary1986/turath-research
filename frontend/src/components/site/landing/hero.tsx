import Image from "next/image";
import { getTranslations } from "next-intl/server";
import { Megaphone } from "lucide-react";
import { Link } from "@/i18n/routing";
import { Button } from "@/components/ui/button";
import { Bidi } from "@/components/i18n/bidi";
import type { LocalizedText } from "@/lib/i18n/text-lang";
import { PortalEntryButton } from "../portal-entry-button";

interface HeroProps {
  locale: string;
  title: LocalizedText | null;
  subtitle: LocalizedText | null;
}

const HERO_TITLE_CLASS = "text-2xl leading-tight font-bold text-balance sm:text-3xl md:text-5xl";

/** Navy hero with the cuneiform texture, the crest beside the title and one primary action. */
export async function Hero({ locale, title, subtitle }: HeroProps) {
  const t = await getTranslations({ locale, namespace: "landing.hero" });
  const ts = await getTranslations({ locale, namespace: "site" });

  return (
    <section aria-labelledby="hero-title" className="hero-cuneiform text-primary-foreground">
      <div className="mx-auto max-w-6xl px-4 py-12 md:py-20">
        <div className="flex items-start gap-4 md:items-center md:gap-10">
          <div className="min-w-0 flex-1">
            {title ? (
              <Bidi as="h1" id="hero-title" block text={title.text} lang={title.lang} className={HERO_TITLE_CLASS} />
            ) : (
              <h1 id="hero-title" className={HERO_TITLE_CLASS}>{ts("meta.homeTitle")}</h1>
            )}
            {subtitle && (
              <Bidi as="p" block text={subtitle.text} lang={subtitle.lang}
                className="mt-4 max-w-2xl text-base leading-relaxed text-pretty whitespace-pre-line text-white/90 md:text-lg" />
            )}
          </div>
          {/* Fixed box per breakpoint (crest is 634×844) so the image never shifts the layout. */}
          <Image
            src="/brand/crest.png"
            width={634}
            height={844}
            alt={ts("logoAlt")}
            loading="eager"
            fetchPriority="high"
            sizes="(min-width: 768px) 176px, 80px"
            className="order-first h-auto w-16 shrink-0 drop-shadow-lg select-none sm:w-20 md:order-none md:w-44"
          />
        </div>
        <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-center">
          <PortalEntryButton size="lg" tone="onDark" signInLabel={t("primaryCta")} className="w-full sm:w-auto" />
          <Button
            size="lg"
            variant="ghost"
            className="h-12 w-full px-5 text-base text-white underline-offset-4 hover:bg-white/10 hover:text-white sm:w-auto"
            nativeButton={false} render={<Link href="/news" />}
          >
            <Megaphone className="size-4" aria-hidden />
            {t("secondaryCta")}
          </Button>
        </div>
        <p className="mt-3 text-sm text-white/80">{t("signInHint")}</p>
      </div>
    </section>
  );
}
