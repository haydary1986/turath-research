import { getLocale, getTranslations } from "next-intl/server";
import { Link } from "@/i18n/routing";
import { JOURNAL_PATHS } from "@/lib/journals/lists";

/** A slug that is not (or no longer) in the lists. */
export default async function EntryNotFound() {
  const locale = await getLocale();
  const t = await getTranslations({ locale, namespace: "journals.notFound" });
  return (
    <div className="mx-auto max-w-2xl px-4 py-20 text-center">
      <h1 className="text-2xl font-bold text-primary">{t("title")}</h1>
      <p className="mt-3 text-muted-foreground">{t("body")}</p>
      <Link href={JOURNAL_PATHS.list} className="mt-6 inline-block font-semibold text-primary hover:text-accent hover:underline">
        {t("back")}
      </Link>
    </div>
  );
}
