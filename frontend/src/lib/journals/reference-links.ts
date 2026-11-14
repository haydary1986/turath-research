/**
 * The trusted external references researchers check a journal against, grouped as the
 * University of Baghdad's predatory-journals page groups them. The portal links out to
 * these lists instead of copying them.
 */
export interface ReferenceLink {
  key: string;
  href: string;
}

export interface ReferenceGroup {
  key: "predatory" | "hijacked" | "ministry";
  links: ReferenceLink[];
}

export const REFERENCE_GROUPS: ReferenceGroup[] = [
  {
    key: "predatory",
    links: [
      { key: "beall", href: "https://beallslist.net/" },
      { key: "predatoryJournals", href: "https://www.predatoryjournals.org/home" },
    ],
  },
  {
    key: "hijacked",
    links: [{ key: "retractionWatch", href: "https://retractionwatch.com/the-retraction-watch-hijacked-journal-checker/" }],
  },
  {
    key: "ministry",
    links: [{ key: "ministryPortal", href: "https://jor.rdd.edu.iq/" }],
  },
];
