import Image from "next/image";
import { getTranslations } from "next-intl/server";
import { Quote, UserRound } from "lucide-react";
import { PUBLIC_API, type SiteDirector } from "@/lib/site/types";
import { cn } from "@/lib/utils";
import { Bidi } from "@/components/i18n/bidi";
import { Localized } from "@/components/i18n/localized";
import { pickLocalized } from "@/lib/i18n/text-lang";
import { RichText } from "../rich-text";
import { Section } from "./section";

/** The director's message; hidden while neither a name nor a message is set. */
export async function DirectorMessage({ locale, director }: { locale: string; director: SiteDirector }) {
  if (!director.name && !director.message && !director.name_en && !director.message_en) return null;
  const t = await getTranslations({ locale, namespace: "landing.director" });
  const title = pickLocalized(director.title, director.title_en, locale);
  const name = pickLocalized(director.name, director.name_en, locale);
  const message = pickLocalized(director.message, director.message_en, locale);

  return (
    <Section id="director" title={title ? <Bidi text={title.text} lang={title.lang} /> : t("defaultTitle")}>
      <figure className={cn("grid gap-6 rounded-2xl border bg-card p-6 md:p-8", message && "md:grid-cols-[auto_1fr]")}>
        <div className="flex items-center gap-4 md:flex-col md:items-center md:text-center">
          {director.photo_id ? (
            <Image
              src={PUBLIC_API.directorPhoto(director.photo_id)}
              alt={t("photoAlt", { name: name?.text ?? "" })}
              width={160}
              height={160}
              unoptimized
              className="size-24 rounded-2xl object-cover ring-4 ring-primary/10 md:size-40"
            />
          ) : (
            <span className="flex size-24 items-center justify-center rounded-2xl bg-primary/5 text-primary md:size-40" aria-hidden>
              <UserRound className="size-12" />
            </span>
          )}
          <figcaption className="min-w-0">
            {name && <Bidi as="p" block text={name.text} lang={name.lang} className="font-bold text-primary" />}
            <Localized as="p" block ar={director.position} en={director.position_en} locale={locale} className="text-sm text-muted-foreground" />
          </figcaption>
        </div>
        {message && (
          <blockquote className="relative">
            <Quote className="absolute -top-1 -start-1 size-8 text-accent/20" aria-hidden />
            <RichText text={message.text} lang={message.lang} className="ps-8 text-base leading-loose" />
          </blockquote>
        )}
      </figure>
    </Section>
  );
}
