import { getTranslations } from "next-intl/server";
import { ShieldAlert } from "lucide-react";
import { Link } from "@/i18n/routing";
import { Button } from "@/components/ui/button";
import { JOURNAL_PATHS } from "@/lib/journals/lists";

/** Internal links from the homepage to the public journal guide and tools. */
export async function JournalsPromo({ locale }: { locale: string }) {
  const t = await getTranslations({ locale, namespace: "landing.journalsPromo" });
  return (
    <section aria-labelledby="journals-promo-title" className="mx-auto max-w-6xl px-4 pt-14">
      <div className="flex flex-col gap-6 rounded-2xl border border-accent/20 bg-accent/5 p-6 md:flex-row md:items-center md:p-8">
        <span className="flex size-14 shrink-0 items-center justify-center rounded-2xl bg-accent text-accent-foreground">
          <ShieldAlert className="size-7" aria-hidden />
        </span>
        <div className="min-w-0 flex-1">
          <h2 id="journals-promo-title" className="text-xl font-bold text-primary md:text-2xl">{t("title")}</h2>
          <p className="mt-2 max-w-3xl leading-relaxed text-muted-foreground">{t("body")}</p>
          <div className="mt-4 flex flex-wrap gap-2">
            <Button nativeButton={false} render={<Link href={JOURNAL_PATHS.guide} />}>
              <ShieldAlert className="size-4" aria-hidden />
              {t("guideCta")}
            </Button>
          </div>
        </div>
      </div>
    </section>
  );
}
