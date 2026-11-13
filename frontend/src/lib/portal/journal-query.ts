export interface JournalQuery {
  issn?: string;
  issn_online?: string;
  name?: string;
  publisher?: string;
  url?: string;
}

/** Keep equal to journalcheck.MinNameSearchRunes on the backend. */
export const MIN_NAME_SEARCH_LENGTH = 4;

const ISSN_LENGTH = 8;
const ARABIC_INDIC_ZERO = 0x0660;
const PERSIAN_ZERO = 0x06f0;

/** Rewrites Arabic-Indic and Persian digits as ASCII. */
export function toLatinDigits(value: string): string {
  return value.replace(/[٠-٩۰-۹]/g, (digit) => {
    const code = digit.charCodeAt(0);
    return String(code - (code >= PERSIAN_ZERO ? PERSIAN_ZERO : ARABIC_INDIC_ZERO));
  });
}

function completeISSN(value: string | undefined): string {
  const compact = toLatinDigits(value ?? "").replace(/[^0-9Xx]/g, "");
  return compact.length === ISSN_LENGTH ? compact : "";
}

/** Mirrors the backend's RefKey closely enough to measure a name the same way. */
export function nameSearchLength(name: string): number {
  const folded = name
    .toLowerCase()
    .replace(/&/g, " and ")
    .replace(/[ً-ْ]/g, "")
    .replace(/[\p{P}\p{S}]+/gu, " ")
    .replace(/\s+/g, " ")
    .trim()
    .replace(/^the /, "");
  return [...folded].length;
}

/** Builds the query string, or "" while the input is still too partial to check. */
export function journalCheckParams(query: JournalQuery): string {
  const issn = completeISSN(query.issn);
  const online = completeISSN(query.issn_online);
  const name = query.name?.trim() ?? "";
  const url = query.url?.trim() ?? "";
  const publisher = query.publisher?.trim() ?? "";
  const nameUsable = nameSearchLength(name) >= MIN_NAME_SEARCH_LENGTH;
  if (!issn && !online && !nameUsable && publisher.length < MIN_NAME_SEARCH_LENGTH && !url) return "";

  const params = new URLSearchParams();
  if (issn) params.set("issn", issn);
  if (online) params.set("issn_online", online);
  if (name) params.set("name", name);
  if (publisher) params.set("publisher", publisher);
  if (url) params.set("url", url);
  return params.toString();
}

/**
 * Reads the search box as an ISSN or a journal name. null means nothing checkable:
 * a name too short to search, or digits that are not yet a whole ISSN.
 */
export function parseJournalSearch(text: string): JournalQuery | null {
  const issn = completeISSN(text);
  const looksNumeric = /^[\d\s\p{Pd}‎‏Xx]*$/u.test(toLatinDigits(text));
  if (looksNumeric) return issn ? { issn } : null;
  return nameSearchLength(text) >= MIN_NAME_SEARCH_LENGTH ? { name: text.trim() } : null;
}
