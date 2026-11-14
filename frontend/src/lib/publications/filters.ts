/** Filters of the publications list, kept in the URL so a filtered view can be shared. */
export interface PublicationFilters {
  q: string;
  author: string;
  journal: string;
  year_from: string;
  year_to: string;
  type: string;
  college_id: string;
  department_id: string;
  sort: PublicationSort;
  page: number;
}

export const PUBLICATION_SORTS = ["newest", "oldest", "cited", "title"] as const;
export type PublicationSort = (typeof PUBLICATION_SORTS)[number];

export const PUBLICATION_TYPES = [
  "journal_article", "conference_paper", "book_chapter", "book", "preprint", "thesis", "patent", "report",
] as const;

export const PAGE_SIZE = 20;

const TEXT_KEYS = ["q", "author", "journal", "year_from", "year_to", "type", "college_id", "department_id"] as const;

export const EMPTY_FILTERS: PublicationFilters = {
  q: "", author: "", journal: "", year_from: "", year_to: "", type: "",
  college_id: "", department_id: "", sort: "newest", page: 1,
};

const YEAR = /^\d{4}$/;

/** Reads filters from a query string, dropping anything malformed. */
export function filtersFromParams(params: URLSearchParams): PublicationFilters {
  const pick = (key: string): string => params.get(key)?.trim() ?? "";
  const sort = pick("sort");
  const page = Number(pick("page"));
  const type = pick("type");
  return {
    q: pick("q"),
    author: pick("author"),
    journal: pick("journal"),
    year_from: YEAR.test(pick("year_from")) ? pick("year_from") : "",
    year_to: YEAR.test(pick("year_to")) ? pick("year_to") : "",
    type: (PUBLICATION_TYPES as readonly string[]).includes(type) ? type : "",
    college_id: pick("college_id"),
    department_id: pick("department_id"),
    sort: (PUBLICATION_SORTS as readonly string[]).includes(sort) ? (sort as PublicationSort) : "newest",
    page: Number.isInteger(page) && page > 0 ? page : 1,
  };
}

/** Query string for the URL: only the values that differ from the defaults. */
export function filtersToParams(filters: PublicationFilters): URLSearchParams {
  const params = new URLSearchParams();
  for (const key of TEXT_KEYS) {
    const value = filters[key].trim();
    if (value) params.set(key, value);
  }
  if (filters.sort !== "newest") params.set("sort", filters.sort);
  if (filters.page > 1) params.set("page", String(filters.page));
  return params;
}

/** Query string for the API, including the page size. */
export function filtersToApiQuery(filters: PublicationFilters): string {
  const params = filtersToParams(filters);
  params.set("sort", filters.sort);
  params.set("page", String(filters.page));
  params.set("limit", String(PAGE_SIZE));
  return params.toString();
}

/** Changing any filter returns to the first page; a college change clears a department of another college. */
export function withFilter(
  filters: PublicationFilters,
  patch: Partial<Omit<PublicationFilters, "page">>
): PublicationFilters {
  const next = { ...filters, ...patch, page: 1 };
  if (patch.college_id !== undefined && patch.college_id !== filters.college_id) next.department_id = "";
  return next;
}

/** Number of active filters (sort and page are not filters). */
export function activeFilterCount(filters: PublicationFilters): number {
  return TEXT_KEYS.filter((key) => filters[key].trim() !== "").length;
}
