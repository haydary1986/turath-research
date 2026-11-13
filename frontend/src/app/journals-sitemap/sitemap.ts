import type { MetadataRoute } from "next";
import { REVALIDATE, backendGet } from "@/lib/site/server-api";
import { JOURNAL_PATHS, LIST_TYPES, PUBLIC_JOURNALS_API, isListType, type JournalSitemapEntry } from "@/lib/journals/lists";
import { localizedEntries } from "@/lib/seo/sitemap";

// Rendered per request: the backend is not reachable while the image is built.
export const dynamic = "force-dynamic";

const PAGE = 5000;
/** Two URLs (ar, en) per entry keeps one list well under the 50,000-URL limit. */
const MAX_ENTRIES = 24000;

/** One sitemap per reference list: /journals-sitemap/sitemap/<list>.xml */
export async function generateSitemaps(): Promise<{ id: string }[]> {
  return LIST_TYPES.map((id) => ({ id }));
}

async function loadSlugs(type: string): Promise<JournalSitemapEntry[]> {
  const out: JournalSitemapEntry[] = [];
  for (let page = 1; out.length < MAX_ENTRIES; page++) {
    const params = new URLSearchParams({ type, page: String(page), limit: String(PAGE) });
    const result = await backendGet<JournalSitemapEntry[]>(`${PUBLIC_JOURNALS_API.sitemap}?${params}`, REVALIDATE.fresh);
    const items = result?.data ?? [];
    out.push(...items);
    if (items.length < PAGE) break;
  }
  return out.slice(0, MAX_ENTRIES);
}

export default async function sitemap(props: { id: Promise<string> }): Promise<MetadataRoute.Sitemap> {
  const id = await props.id;
  if (!isListType(id)) return [];
  const entries = await loadSlugs(id);
  return entries.flatMap((entry) =>
    localizedEntries({
      path: JOURNAL_PATHS.entry(entry.slug),
      lastModified: entry.fetched_at,
      changeFrequency: "weekly",
      priority: 0.6,
    })
  );
}
