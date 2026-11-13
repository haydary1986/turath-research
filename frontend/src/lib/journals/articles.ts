import { ARTICLES_AR } from "./articles-ar";
import { ARTICLES_EN } from "./articles-en";
import type { Article, ArticleKey, ArticleSet } from "./article-types";

export * from "./article-types";

/** The guide articles in the page's language (Arabic is the default). */
export function articlesFor(locale: string): ArticleSet {
  return locale === "en" ? ARTICLES_EN : ARTICLES_AR;
}

export function getArticle(locale: string, key: ArticleKey): Article {
  return articlesFor(locale)[key];
}
