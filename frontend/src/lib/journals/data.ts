import { REVALIDATE, backendGet, backendGetOptional, type BackendResult } from "@/lib/site/server-api";
import {
  INDEX_PAGE_SIZE, LETTER_OTHER, PUBLIC_JOURNALS_API, isEntrySlug,
  type JournalEntryDetail, type JournalLetter, type JournalListStatus, type ListType, type PublicJournalEntry,
} from "./lists";

/** Server-side loaders for the public journal pages. */

export async function loadListStatuses(): Promise<JournalListStatus[]> {
  return (await backendGetOptional<JournalListStatus[]>(PUBLIC_JOURNALS_API.status, REVALIDATE.lists))?.data ?? [];
}

/** One entry with its related entries; null when the slug is not (or no longer) listed. */
export async function loadEntry(slug: string): Promise<JournalEntryDetail | null> {
  if (!isEntrySlug(slug)) return null;
  return (await backendGet<JournalEntryDetail>(PUBLIC_JOURNALS_API.entry(slug), REVALIDATE.entry))?.data ?? null;
}

export async function loadLetters(type: ListType): Promise<JournalLetter[]> {
  return (await backendGet<JournalLetter[]>(PUBLIC_JOURNALS_API.letters(type), REVALIDATE.lists))?.data ?? [];
}

export async function loadIndexPage(
  type: ListType,
  letter: string | undefined,
  page: number
): Promise<BackendResult<PublicJournalEntry[]>> {
  const params = new URLSearchParams({ type, page: String(page), limit: String(INDEX_PAGE_SIZE) });
  if (letter) params.set("letter", letter === LETTER_OTHER ? LETTER_OTHER : letter);
  const result = await backendGet<PublicJournalEntry[]>(`${PUBLIC_JOURNALS_API.entries}?${params}`, REVALIDATE.lists);
  return result ?? { data: [], meta: { total: 0, page, limit: INDEX_PAGE_SIZE } };
}
