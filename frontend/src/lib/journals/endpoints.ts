/**
 * Where the journal check and the list browser read from. The portal uses the
 * signed-in endpoints; the public site uses the anonymous, rate-limited ones
 * (whose list entries also carry a slug for the entry page).
 */
export interface JournalEndpoints {
  check: string;
  status: string;
  list: string;
  /** True for the public endpoints: entries have slugs and link to their pages. */
  public: boolean;
}

export const PORTAL_JOURNAL_ENDPOINTS: JournalEndpoints = {
  check: "/api/v1/screening/check",
  status: "/api/v1/screening/status",
  list: "/api/v1/screening/reference-list",
  public: false,
};

export const PUBLIC_JOURNAL_ENDPOINTS: JournalEndpoints = {
  check: "/api/v1/public/journals/check",
  status: "/api/v1/public/journals/status",
  list: "/api/v1/public/journals/entries",
  public: true,
};
