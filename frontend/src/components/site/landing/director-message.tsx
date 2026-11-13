import Image from "next/image";
import { getTranslations } from "next-intl/server";
import { Quote, UserRound } from "lucide-react";
import { PUBLIC_API, type SiteDirector } from "@/lib/site/types";
import { cn } from "@/lib/utils";
import { RichText } from "../rich-text";
import { Section } from "./section";

/** The director's message; hidden while neither a name nor a message is set. */
export async function DirectorMessage({ locale, director }: { locale: string; director: SiteDirector }) {
  if (!director.name && !director.message) return null;
  const t = await getTranslations({ locale, namespace: "landing.director" });

  return (
    <Section id="director" title={director.title || t("defaultTitle")}>
      <figure className={cn("grid gap-6 rounded-2xl border bg-card p-6 md:p-8", director.message && "md:grid-cols-[auto_1fr]")}>
        <div className="flex items-center gap-4 md:flex-col md:items-center md:text-center">
          {director.photo_id ? (
            <Image
              src={PUBLIC_API.directorPhoto(director.photo_id)}
              alt={t("photoAlt", { name: director.name })}
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
            {director.name && <p className="font-bold text-primary">{director.name}</p>}
            {director.position && <p className="text-sm text-muted-foreground">{director.position}</p>}
          </figcaption>
        </div>
        {director.message && (
          <blockquote className="relative">
            <Quote className="absolute -top-1 -start-1 size-8 text-accent/20" aria-hidden />
            <RichText text={director.message} className="ps-8 text-base leading-loose" />
          </blockquote>
        )}
      </figure>
    </Section>
  );
}
