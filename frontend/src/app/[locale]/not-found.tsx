import { getLocale, getTranslations } from "next-intl/server";
import { Link } from "@/i18n/routing";

/** 404 inside a locale: rendered within the locale layout, so it has the page's lang/dir and language. */
export default async function LocaleNotFound() {
  const locale = await getLocale();
  const t = await getTranslations({ locale, namespace: "notFound" });
  return (
    <main className="mx-auto max-w-2xl px-4 py-24 text-center">
      <h1 className="text-2xl font-bold text-primary">{t("title")}</h1>
      <p className="mt-3 text-muted-foreground">{t("body")}</p>
      <Link href="/" className="mt-6 inline-block font-semibold text-primary hover:text-accent hover:underline">
        {t("home")}
      </Link>
    </main>
  );
}
