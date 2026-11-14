import type { MetadataRoute } from "next";
import { ARTICLES_UPDATED, TOPIC_SLUGS } from "@/lib/journals/articles";
import { JOURNAL_PATHS } from "@/lib/journals/lists";
import { localizedEntries, type SitemapPage } from "@/lib/seo/sitemap";

// Rendered per request: the backend is not reachable while the image is built.
export const dynamic = "force-dynamic";

/** The public site's pages. */
export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const pages: SitemapPage[] = [
    { path: "/", changeFrequency: "daily", priority: 1 },
    { path: "/news", changeFrequency: "daily", priority: 0.8 },
    { path: JOURNAL_PATHS.hub, changeFrequency: "monthly", priority: 0.8, lastModified: ARTICLES_UPDATED },
    { path: JOURNAL_PATHS.guide, changeFrequency: "monthly", priority: 0.9, lastModified: ARTICLES_UPDATED },
    { path: JOURNAL_PATHS.check, changeFrequency: "monthly", priority: 0.9 },
    ...TOPIC_SLUGS.map((slug) => ({
      path: JOURNAL_PATHS.topic(slug), changeFrequency: "monthly" as const, priority: 0.8, lastModified: ARTICLES_UPDATED,
    })),
  ];

  return pages.flatMap(localizedEntries);
}
