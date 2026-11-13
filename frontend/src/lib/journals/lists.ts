/** Public journal pages: reference-list types, entry shapes and site paths. */

export const LIST_TYPES = ["predatory_journal", "predatory_publisher", "hijacked"] as const;
export type ListType = (typeof LIST_TYPES)[number];

/** URL segment of each list's A–Z index. */
export const LIST_SEGMENT: Record<ListType, string> = {
  predatory_journal: "journals",
  predatory_publisher: "publishers",
  hijacked: "hijacked",
};

export const LETTERS = "abcdefghijklmnopqrstuvwxyz".split("");
export const LETTER_OTHER = "other";
export const INDEX_PAGE_SIZE = 100;

export interface PublicJournalEntry {
  slug: string;
  name: string;
  list_type: ListType;
  hosts: string[];
  authentic_name: string | null;
  authentic_url: string | null;
  source: string;
  fetched_at: string;
}

export interface JournalEntryLink {
  slug: string;
  name: string;
  list_type: ListType;
}

export interface JournalEntryDetail {
  entry: PublicJournalEntry;
  same_host: JournalEntryLink[];
  neighbours: JournalEntryLink[];
  list_refreshed_at: string | null;
}

export interface JournalLetter {
  letter: string;
  count: number;
}

export interface JournalListStatus {
  list_type: ListType;
  count: number;
  last_attempt_at: string | null;
  last_success_at: string | null;
  last_failed: boolean;
}

export interface JournalSitemapEntry {
  slug: string;
  fetched_at: string;
}

export const PUBLIC_JOURNALS_API = {
  status: "/api/v1/public/journals/status",
  check: "/api/v1/public/journals/check",
  entries: "/api/v1/public/journals/entries",
  entry: (slug: string): string => `/api/v1/public/journals/entries/${encodeURIComponent(slug)}`,
  letters: (type: ListType): string => `/api/v1/public/journals/letters?type=${type}`,
  sitemap: "/api/v1/public/journals/sitemap",
} as const;

/** The upstream lists the refresher downloads (credited on every entry page). */
export const SOURCE_LIST_URL = "https://github.com/stop-predatory-journals/stop-predatory-journals.github.io";

export function isListType(value: string | undefined | null): value is ListType {
  return !!value && (LIST_TYPES as readonly string[]).includes(value);
}

export function listTypeFromSegment(segment: string): ListType | null {
  const found = LIST_TYPES.find((type) => LIST_SEGMENT[type] === segment);
  return found ?? null;
}

export function isLetter(value: string | undefined): value is string {
  return !!value && (value === LETTER_OTHER || LETTERS.includes(value));
}

/** Site paths (locale-less) of the journal pages. */
export const JOURNAL_PATHS = {
  hub: "/journals",
  guide: "/journals/predatory",
  check: "/journals/check",
  list: "/journals/predatory/list",
  topic: (slug: string): string => `/journals/guide/${slug}`,
  entry: (slug: string): string => `/journals/predatory/${slug}`,
  index: (type: ListType, letter?: string, page = 1): string => {
    const base = `/journals/predatory/list/${LIST_SEGMENT[type]}`;
    const params = new URLSearchParams();
    if (letter) params.set("letter", letter);
    if (page > 1) params.set("page", String(page));
    const query = params.toString();
    return query ? `${base}?${query}` : base;
  },
} as const;

/** Slugs come from the backend; reject anything else before calling it. */
export function isEntrySlug(slug: string): boolean {
  return /^[a-z0-9][a-z0-9-]{0,100}$/.test(slug);
}
