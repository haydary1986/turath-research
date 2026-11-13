import { getRequestConfig } from "next-intl/server";
import { routing } from "./routing";

export default getRequestConfig(async ({ requestLocale }) => {
  let locale = await requestLocale;

  if (!locale || !routing.locales.includes(locale as "ar" | "en")) {
    locale = routing.defaultLocale;
  }

  // The public site's strings live in their own file under separate top-level
  // namespaces (landing, journals, siteEditor), so the two never collide.
  const [main, publicSite] = await Promise.all([
    import(`../messages/${locale}.json`),
    import(`../messages/public/${locale}.json`),
  ]);

  return {
    locale,
    messages: { ...main.default, ...publicSite.default },
  };
});
