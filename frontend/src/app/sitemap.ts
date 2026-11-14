import type { MetadataRoute } from "next";
import { JOURNAL_PATHS } from "@/lib/journals/lists";
import { localizedEntries, type SitemapPage } from "@/lib/seo/sitemap";

// Rendered per request: the backend is not reachable while the image is built.
export const dynamic = "force-dynamic";

/** The public site's pages. */
export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const pages: SitemapPage[] = [
    { path: "/", changeFrequency: "daily", priority: 1 },
    { path: "/news", changeFrequency: "daily", priority: 0.8 },
    { path: JOURNAL_PATHS.guide, changeFrequency: "monthly", priority: 0.9 },
  ];

  return pages.flatMap(localizedEntries);
}
