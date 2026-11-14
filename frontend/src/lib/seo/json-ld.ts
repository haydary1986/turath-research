import type { ReactNode } from "react";
import { SITE_URL, absoluteUrl } from "./site.ts";

/** JSON-LD builders (schema.org) for the public pages. */

export type JsonLdObject = Record<string, unknown>;

/** Serialises JSON-LD for a <script> tag; "<" is escaped so content cannot close the tag. */
export function serializeJsonLd(data: JsonLdObject | JsonLdObject[]): string {
  return JSON.stringify(data).replace(/</g, "\\u003c");
}

export interface OrganizationNames {
  university: string;
  department: string;
  description: string;
}

/** The university and its scientific-affairs department. */
export function organizationLd(locale: string, names: OrganizationNames): JsonLdObject {
  return {
    "@context": "https://schema.org",
    "@type": "CollegeOrUniversity",
    "@id": "https://uoturath.edu.iq/#organization",
    name: names.university,
    url: "https://uoturath.edu.iq",
    logo: `${SITE_URL}/brand/crest-512.png`,
    department: {
      "@type": "Organization",
      "@id": `${SITE_URL}/#department`,
      name: names.department,
      url: absoluteUrl(locale, "/"),
      description: names.description,
    },
  };
}

export interface Crumb {
  name: string;
  /** Locale-less path. */
  path: string;
  /** The name is database text (e.g. a journal name), laid out by its own script. */
  data?: boolean;
  /** Visible label when it needs markup (the structured data keeps `name`). */
  label?: ReactNode;
}

export function breadcrumbLd(locale: string, crumbs: Crumb[]): JsonLdObject {
  return {
    "@context": "https://schema.org",
    "@type": "BreadcrumbList",
    itemListElement: crumbs.map((crumb, i) => ({
      "@type": "ListItem",
      position: i + 1,
      name: crumb.name,
      item: absoluteUrl(locale, crumb.path),
    })),
  };
}

export interface FaqEntry {
  question: string;
  answer: string;
}

export function faqLd(entries: FaqEntry[]): JsonLdObject {
  return {
    "@context": "https://schema.org",
    "@type": "FAQPage",
    mainEntity: entries.map((e) => ({
      "@type": "Question",
      name: e.question,
      acceptedAnswer: { "@type": "Answer", text: e.answer },
    })),
  };
}

export interface ArticleInput {
  locale: string;
  path: string;
  headline: string;
  description: string;
  publisher: string;
  dateModified: string;
}

export function articleLd(input: ArticleInput): JsonLdObject {
  return {
    "@context": "https://schema.org",
    "@type": "Article",
    headline: input.headline,
    description: input.description,
    inLanguage: input.locale,
    mainEntityOfPage: absoluteUrl(input.locale, input.path),
    dateModified: input.dateModified,
    image: `${SITE_URL}/brand/og-image.png`,
    author: { "@type": "Organization", name: input.publisher, url: absoluteUrl(input.locale, "/") },
    publisher: {
      "@type": "Organization",
      name: input.publisher,
      logo: { "@type": "ImageObject", url: `${SITE_URL}/brand/crest-512.png` },
    },
  };
}

export interface ListItemInput {
  name: string;
  path: string;
}

export function itemListLd(locale: string, name: string, items: ListItemInput[], offset = 0): JsonLdObject {
  return {
    "@context": "https://schema.org",
    "@type": "ItemList",
    name,
    numberOfItems: items.length,
    itemListElement: items.map((item, i) => ({
      "@type": "ListItem",
      position: offset + i + 1,
      name: item.name,
      url: absoluteUrl(locale, item.path),
    })),
  };
}
