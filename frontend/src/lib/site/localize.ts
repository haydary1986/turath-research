import { pickLocalized, type TextLang } from "../i18n/text-lang.ts";
import type { SiteFAQItem } from "./types.ts";

/** A FAQ entry of the site document in the page's language (Arabic where English is missing). */
export interface LocalizedFaqItem {
  question: string;
  answer: string;
  questionLang: TextLang;
  answerLang: TextLang;
}

/** Picks each FAQ entry's question and answer for a locale; entries without text are dropped. */
export function localizeFaq(items: SiteFAQItem[], locale: string): LocalizedFaqItem[] {
  return items.flatMap((item) => {
    const question = pickLocalized(item.question, item.question_en, locale);
    const answer = pickLocalized(item.answer, item.answer_en, locale);
    if (!question || !answer) return [];
    return [{ question: question.text, answer: answer.text, questionLang: question.lang, answerLang: answer.lang }];
  });
}
