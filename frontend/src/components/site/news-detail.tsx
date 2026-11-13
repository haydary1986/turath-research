"use client";

import { useLocale, useTranslations } from "next-intl";
import { ArrowLeft, ArrowRight, Download, FileText } from "lucide-react";
import { Link } from "@/i18n/routing";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { PUBLIC_API, formatFileSize, type PublicAnnouncement } from "@/lib/site/types";
import { AnnouncementMeta } from "./announcement-meta";
import { RichText } from "./rich-text";
import { usePublicResource } from "./use-public-resource";

const NOT_FOUND = "Announcement not found";

/** One published announcement with its body and attachment. */
export function NewsDetail({ id }: { id: string }) {
  const t = useTranslations("site");
  const locale = useLocale();
  const Back = locale === "ar" ? ArrowRight : ArrowLeft;
  const { data, error, loading, reload } = usePublicResource<PublicAnnouncement>(PUBLIC_API.announcement(id));

  const back = (
    <Link href="/news" className="inline-flex items-center gap-1 text-sm font-medium text-primary hover:text-accent">
      <Back className="size-4" aria-hidden />
      {t("news.back")}
    </Link>
  );

  if (loading) {
    return (
      <div className="space-y-4" aria-hidden>
        <Skeleton className="h-5 w-40" />
        <Skeleton className="h-9 w-3/4" />
        <Skeleton className="h-40 w-full" />
      </div>
    );
  }
  if (error || !data) {
    return (
      <div className="space-y-4">
        {back}
        {error === NOT_FOUND ? (
          <Alert variant="neutral">{t("news.notFound")}</Alert>
        ) : (
          <Alert variant="warning">
            {t("home.loadError")}{" "}
            <button type="button" onClick={reload} className="font-semibold underline">{t("home.retry")}</button>
          </Alert>
        )}
      </div>
    );
  }

  const attachment = data.attachment;
  const isImage = attachment?.mime_type.startsWith("image/");
  return (
    <article className="space-y-6">
      {back}
      <header className="space-y-3">
        <AnnouncementMeta item={data} />
        <h1 className="text-2xl leading-snug font-bold text-primary md:text-3xl">{data.title}</h1>
      </header>
      {data.body && <RichText text={data.body} className="text-base" />}
      {attachment && (
        <section aria-label={t("news.attachment")} className="space-y-3 rounded-xl border bg-card p-4">
          {isImage && (
            // eslint-disable-next-line @next/next/no-img-element -- served by the API, sizes vary
            <img
              src={PUBLIC_API.file(attachment.id)}
              alt={data.title}
              className="mx-auto max-h-[70vh] w-auto rounded-lg"
              loading="lazy"
            />
          )}
          <div className="flex flex-wrap items-center justify-between gap-3">
            <p className="flex min-w-0 items-center gap-2 text-sm">
              <FileText className="size-5 shrink-0 text-accent" aria-hidden />
              <span className="truncate font-medium" dir="auto">{attachment.name}</span>
              <span className="shrink-0 text-muted-foreground" dir="ltr">{formatFileSize(attachment.size_bytes, locale)}</span>
            </p>
            <Button render={<a href={PUBLIC_API.file(attachment.id)} download={attachment.name} />}>
              <Download className="size-4" aria-hidden />
              {t("news.download")}
            </Button>
          </div>
        </section>
      )}
    </article>
  );
}
