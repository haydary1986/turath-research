/** Site paths (locale-less) of the public journal pages. */
export const JOURNAL_PATHS = {
  hub: "/journals",
  guide: "/journals/predatory",
  check: "/journals/check",
  topic: (slug: string): string => `/journals/guide/${slug}`,
} as const;
