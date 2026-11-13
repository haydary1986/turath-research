import type { MetadataRoute } from "next";
import { SEO_LOCALES, absoluteUrl } from "./site";

type SitemapEntry = MetadataRoute.Sitemap[number];

export interface SitemapPage {
  /** Locale-less path. */
  path: string;
  lastModified?: string | Date;
  changeFrequency?: SitemapEntry["changeFrequency"];
  priority?: number;
}

/** One sitemap URL per locale, each listing its hreflang alternates. */
export function localizedEntries(page: SitemapPage): MetadataRoute.Sitemap {
  const languages = Object.fromEntries(SEO_LOCALES.map((l) => [l, absoluteUrl(l, page.path)]));
  return SEO_LOCALES.map((locale) => ({
    url: absoluteUrl(locale, page.path),
    lastModified: page.lastModified,
    changeFrequency: page.changeFrequency,
    priority: page.priority,
    alternates: { languages },
  }));
}
