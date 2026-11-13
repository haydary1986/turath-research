import assert from "node:assert/strict";
import { test } from "node:test";
import {
  MIN_NAME_SEARCH_LENGTH,
  journalCheckParams,
  nameSearchLength,
  parseJournalSearch,
  toLatinDigits,
} from "./journal-query.ts";

test("toLatinDigits rewrites Arabic-Indic and Persian digits only", () => {
  assert.equal(toLatinDigits("٠٣٧٨-٧٧٨٨"), "0378-7788");
  assert.equal(toLatinDigits("۰۳۷۸۷۷۸۸"), "03787788");
  assert.equal(toLatinDigits("١٢٣x"), "123x");
  assert.equal(toLatinDigits("مجلة 12"), "مجلة 12");
});

test("journalCheckParams sends Arabic-Indic ISSNs as ASCII", () => {
  assert.equal(journalCheckParams({ issn: "٠٣٧٨-٧٧٨٨" }), "issn=03787788");
  assert.equal(journalCheckParams({ issn_online: "۱۸۷۲-۶۱۷۸" }), "issn_online=18726178");
  assert.equal(journalCheckParams({ issn: "٠٣٧٨" }), "");
});

test("the name minimum counts what the backend counts", () => {
  assert.equal(MIN_NAME_SEARCH_LENGTH, 4);
  assert.equal(nameSearchLength("The Cell"), 4);
  assert.equal(nameSearchLength("  c.e.l  "), 5);
  assert.equal(nameSearchLength("مَجَلّة"), 4);
  assert.equal(journalCheckParams({ name: "The BMJ" }), "");
  assert.equal(journalCheckParams({ name: "JAMA" }), "name=JAMA");
});

test("parseJournalSearch tells an ISSN from a name", () => {
  assert.deepEqual(parseJournalSearch(" 0378-7788 "), { issn: "03787788" });
  assert.deepEqual(parseJournalSearch("٠٣٧٨–٧٧٨٨"), { issn: "03787788" });
  assert.deepEqual(parseJournalSearch("‏2049-363x"), { issn: "2049363x" });
  assert.deepEqual(parseJournalSearch("Energy and Buildings"), { name: "Energy and Buildings" });
  assert.deepEqual(parseJournalSearch("مجلة الهندسة"), { name: "مجلة الهندسة" });
});

test("parseJournalSearch refuses input too short to search", () => {
  for (const text of ["", "  ", "abc", "The Ab", "1234", "١٢٣٤-٥", "X"]) {
    assert.equal(parseJournalSearch(text), null, text);
  }
});
