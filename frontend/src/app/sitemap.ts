import type { MetadataRoute } from "next";
import { ARTICLES_UPDATED, TOPIC_SLUGS } from "@/lib/journals/articles";
import { loadLetters } from "@/lib/journals/data";
import { JOURNAL_PATHS, LIST_TYPES } from "@/lib/journals/lists";
import { localizedEntries, type SitemapPage } from "@/lib/seo/sitemap";

// Rendered per request: the backend is not reachable while the image is built.
export const dynamic = "force-dynamic";

/** The site's pages and the journal-list indexes; entry pages are in /journals-sitemap/sitemap/<list>.xml. */
export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const pages: SitemapPage[] = [
    { path: "/", changeFrequency: "daily", priority: 1 },
    { path: "/news", changeFrequency: "daily", priority: 0.8 },
    { path: JOURNAL_PATHS.hub, changeFrequency: "monthly", priority: 0.8, lastModified: ARTICLES_UPDATED },
    { path: JOURNAL_PATHS.guide, changeFrequency: "monthly", priority: 0.9, lastModified: ARTICLES_UPDATED },
    { path: JOURNAL_PATHS.list, changeFrequency: "daily", priority: 0.9 },
    { path: JOURNAL_PATHS.check, changeFrequency: "monthly", priority: 0.9 },
    ...TOPIC_SLUGS.map((slug) => ({
      path: JOURNAL_PATHS.topic(slug), changeFrequency: "monthly" as const, priority: 0.8, lastModified: ARTICLES_UPDATED,
    })),
  ];

  const letterPages = await Promise.all(
    LIST_TYPES.map(async (type): Promise<SitemapPage[]> => {
      const letters = await loadLetters(type).catch(() => []);
      return [
        { path: JOURNAL_PATHS.index(type), changeFrequency: "daily", priority: 0.7 },
        ...letters.map((l) => ({ path: JOURNAL_PATHS.index(type, l.letter), changeFrequency: "daily" as const, priority: 0.5 })),
      ];
    })
  );

  return [...pages, ...letterPages.flat()].flatMap(localizedEntries);
}
