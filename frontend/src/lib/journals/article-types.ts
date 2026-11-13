/** Shapes of the journal-guide articles (server-only content, not shipped as messages). */

export const TOPIC_SLUGS = ["warning-signs", "hijacked-journals", "verify-indexing", "predatory-conferences"] as const;
export type TopicSlug = (typeof TOPIC_SLUGS)[number];

/** The guide hub (/journals/predatory) shares the article shape under this key. */
export const GUIDE_KEY = "guide";
export type ArticleKey = TopicSlug | typeof GUIDE_KEY;

export interface ArticleSection {
  heading: string;
  paragraphs: string[];
  bullets?: string[];
}

export interface FaqItem {
  question: string;
  answer: string;
}

export interface Article {
  /** <title> and <h1>. */
  title: string;
  /** Meta description (≤ 160 characters). */
  description: string;
  /** Short label for navigation and cards. */
  navLabel: string;
  intro: string;
  sections: ArticleSection[];
  faq?: FaqItem[];
}

export type ArticleSet = Record<ArticleKey, Article>;

/** Content revision date, used as Article.dateModified. */
export const ARTICLES_UPDATED = "2026-11-14";

export function isTopicSlug(value: string): value is TopicSlug {
  return (TOPIC_SLUGS as readonly string[]).includes(value);
}
