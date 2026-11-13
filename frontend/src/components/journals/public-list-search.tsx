"use client";

import { PredatoryListBrowser } from "@/components/portal/predatory-list-browser";
import { PUBLIC_JOURNAL_ENDPOINTS } from "@/lib/journals/endpoints";

/** The interactive list search on the public site, on top of the server-rendered indexes. */
export function PublicListSearch() {
  return <PredatoryListBrowser endpoints={PUBLIC_JOURNAL_ENDPOINTS} />;
}
