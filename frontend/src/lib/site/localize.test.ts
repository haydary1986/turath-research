import assert from "node:assert/strict";
import { test } from "node:test";
import { localizeFaq } from "./localize.ts";

const item = { question: "سؤال؟", question_en: "Question?", answer: "جواب", answer_en: "" };

test("English page shows English questions and falls back to the Arabic answer", () => {
  assert.deepEqual(localizeFaq([item], "en"), [
    { question: "Question?", questionLang: "en", answer: "جواب", answerLang: "ar" },
  ]);
});

test("Arabic page shows the Arabic entry", () => {
  assert.deepEqual(localizeFaq([item], "ar"), [
    { question: "سؤال؟", questionLang: "ar", answer: "جواب", answerLang: "ar" },
  ]);
});

test("entries with no text in either language are dropped", () => {
  assert.deepEqual(localizeFaq([{ question: "", question_en: "", answer: "x", answer_en: "" }], "en"), []);
});
