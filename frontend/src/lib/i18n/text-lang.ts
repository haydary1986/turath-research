/**
 * Language and direction of free text (names, titles, notes typed by people or
 * fetched from the database). The UI language says nothing about the language of
 * the data it shows, so every such text is laid out by its own script.
 */

export type TextLang = "ar" | "en";
export type TextDir = "rtl" | "ltr";

// Arabic script blocks: Arabic, Supplement, Extended-A/B, Presentation Forms A/B.
const ARABIC_LETTER = /[؀-ۿݐ-ݿࡰ-ࣿﭐ-﷿ﹰ-﻿]/gu;
// Latin letters, including accented ones.
const LATIN_LETTER = /[A-Za-zÀ-ɏ]/gu;
// Arabic-script letters only (no digits, punctuation or marks), for the "contains Arabic" test.
const ARABIC_ALPHA = /[ء-يٱ-ۓۺ-ۿݐ-ݿࢠ-ࣉﭐ-ﷻﹰ-ﻼ]/u;

/** True when the text holds at least one Arabic letter. */
export function hasArabic(text: string | null | undefined): boolean {
  return !!text && ARABIC_ALPHA.test(text);
}

/**
 * The script a text is mostly written in: "ar" or "en" by letter count, or null
 * when it has no letters at all (numbers, codes, punctuation).
 */
export function detectLang(text: string | null | undefined): TextLang | null {
  if (!text) return null;
  const arabic = text.match(ARABIC_LETTER)?.length ?? 0;
  const latin = text.match(LATIN_LETTER)?.length ?? 0;
  if (arabic === 0 && latin === 0) return null;
  return arabic >= latin ? "ar" : "en";
}

export function dirOf(lang: TextLang): TextDir {
  return lang === "ar" ? "rtl" : "ltr";
}

export function isTextLang(value: string | null | undefined): value is TextLang {
  return value === "ar" || value === "en";
}

/** A text together with the language it is written in. */
export interface LocalizedText {
  text: string;
  lang: TextLang;
}

/**
 * Picks the text of a bilingual field for a page locale: the English value on an
 * English page when it is filled in, otherwise the Arabic (the primary language),
 * otherwise whichever value exists. Returns null when both are empty.
 */
export function pickLocalized(
  ar: string | null | undefined,
  en: string | null | undefined,
  locale: string
): LocalizedText | null {
  const arText = (ar ?? "").trim();
  const enText = (en ?? "").trim();
  if (locale === "en" && enText) return { text: enText, lang: "en" };
  if (arText) return { text: arText, lang: "ar" };
  if (enText) return { text: enText, lang: "en" };
  return null;
}

/**
 * Picks between a record's Arabic and English names (e.g. name_ar / name_en): the
 * one matching the page locale when present, else the other. The language is
 * detected from the text, since "Arabic" columns sometimes hold Latin text.
 */
export function pickName(
  ar: string | null | undefined,
  en: string | null | undefined,
  locale: string
): LocalizedText | null {
  const picked = pickLocalized(ar, en, locale);
  if (!picked) return null;
  return { text: picked.text, lang: detectLang(picked.text) ?? picked.lang };
}
