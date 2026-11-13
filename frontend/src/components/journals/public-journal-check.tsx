"use client";

import { JournalCheckSearch } from "@/components/portal/journal-check-search";
import { PUBLIC_JOURNAL_ENDPOINTS } from "@/lib/journals/endpoints";

/** The journal check on the public site, against the anonymous rate-limited endpoint. */
export function PublicJournalCheck({ initialText }: { initialText?: string }) {
  return <JournalCheckSearch endpoints={PUBLIC_JOURNAL_ENDPOINTS} initialText={initialText} />;
}
