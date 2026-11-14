import assert from "node:assert/strict";
import { test } from "node:test";
import { detectLang, dirOf, hasArabic, pickLocalized, pickName } from "./text-lang.ts";

test("detectLang reads the script of the letters", () => {
  assert.equal(detectLang("أحمد علي"), "ar");
  assert.equal(detectLang("Ahmed Ali"), "en");
  assert.equal(detectLang("د. Ahmed"), "en");
  assert.equal(detectLang("Faculty of Engineering (كلية)"), "en");
  assert.equal(detectLang("كلية الهندسة - IT"), "ar");
  assert.equal(detectLang("10.1000/xyz-123"), "en");
  assert.equal(detectLang("2026-01-01"), null);
  assert.equal(detectLang("  "), null);
  assert.equal(detectLang(null), null);
});

test("hasArabic ignores Arabic digits and punctuation", () => {
  assert.equal(hasArabic("Scientific Affairs"), false);
  assert.equal(hasArabic("١٢٣ ،؟"), false);
  assert.equal(hasArabic("Report تقرير"), true);
});

test("dirOf maps a language to its direction", () => {
  assert.equal(dirOf("ar"), "rtl");
  assert.equal(dirOf("en"), "ltr");
});

test("pickLocalized prefers the page language and falls back to Arabic", () => {
  assert.deepEqual(pickLocalized("قسم", "Department", "en"), { text: "Department", lang: "en" });
  assert.deepEqual(pickLocalized("قسم", "Department", "ar"), { text: "قسم", lang: "ar" });
  assert.deepEqual(pickLocalized("قسم", "  ", "en"), { text: "قسم", lang: "ar" });
  assert.deepEqual(pickLocalized("", "Department", "ar"), { text: "Department", lang: "en" });
  assert.equal(pickLocalized("", null, "en"), null);
});

test("pickName detects the language of the chosen name", () => {
  assert.deepEqual(pickName("Ali Hassan", null, "ar"), { text: "Ali Hassan", lang: "en" });
  assert.deepEqual(pickName("علي حسن", "Ali Hassan", "en"), { text: "Ali Hassan", lang: "en" });
  assert.deepEqual(pickName("علي حسن", "", "en"), { text: "علي حسن", lang: "ar" });
});
