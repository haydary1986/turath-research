import type { Metadata } from "next";

/** Public origin of the site, from SITE_URL (no trailing slash). */
export const SITE_URL = (process.env.SITE_URL || "https://research.uoturath.edu.iq").replace(/\/+$/, "");

export const SEO_LOCALES = ["ar", "en"] as const;
export type SeoLocale = (typeof SEO_LOCALES)[number];

export const OG_IMAGE = { url: "/brand/og-image.png", width: 1200, height: 630 } as const;

const OG_LOCALE: Record<SeoLocale, string> = { ar: "ar_IQ", en: "en_US" };

export function asSeoLocale(locale: string): SeoLocale {
  return locale === "en" ? "en" : "ar";
}

/** Absolute URL of a locale-prefixed site path ("/" is the locale root). */
export function absoluteUrl(locale: string, path: string): string {
  return `${SITE_URL}/${asSeoLocale(locale)}${path === "/" ? "" : path}`;
}

/** Canonical URL plus hreflang alternates (Arabic is the default language). */
export function pageAlternates(locale: string, path: string): NonNullable<Metadata["alternates"]> {
  return {
    canonical: absoluteUrl(locale, path),
    languages: {
      ar: absoluteUrl("ar", path),
      en: absoluteUrl("en", path),
      "x-default": absoluteUrl("ar", path),
    },
  };
}

export interface PageMetadataInput {
  locale: string;
  /** Locale-less path, e.g. "/journals/predatory". */
  path: string;
  title: string;
  description: string;
  siteName: string;
  type?: "website" | "article";
  /** Set when the page title should not get the site-name suffix. */
  absoluteTitle?: boolean;
}

/** Title, description, canonical/hreflang, Open Graph and Twitter card for a public page. */
export function pageMetadata(input: PageMetadataInput): Metadata {
  const { locale, path, title, description, siteName, type = "website", absoluteTitle = false } = input;
  const url = absoluteUrl(locale, path);
  const image = { ...OG_IMAGE, url: `${SITE_URL}${OG_IMAGE.url}`, alt: siteName };
  return {
    title: absoluteTitle ? { absolute: title } : title,
    description,
    alternates: pageAlternates(locale, path),
    openGraph: {
      title,
      description,
      url,
      siteName,
      type,
      locale: OG_LOCALE[asSeoLocale(locale)],
      alternateLocale: [OG_LOCALE[asSeoLocale(locale) === "ar" ? "en" : "ar"]],
      images: [image],
    },
    twitter: { card: "summary_large_image", title, description, images: [image.url] },
    robots: { index: true, follow: true },
  };
}

/** Trims text to a meta-description length on a word boundary. */
export function metaDescription(text: string, max = 160): string {
  const flat = text.replace(/\s+/g, " ").trim();
  if (flat.length <= max) return flat;
  const cut = flat.slice(0, max - 1);
  const space = cut.lastIndexOf(" ");
  return `${(space > max * 0.6 ? cut.slice(0, space) : cut).trim()}…`;
}
