import assert from "node:assert/strict";
import { test } from "node:test";
import { JOURNAL_PATHS, isEntrySlug, isLetter, listTypeFromSegment } from "./lists.ts";

test("index paths carry only the parameters that differ from the defaults", () => {
  assert.equal(JOURNAL_PATHS.index("hijacked"), "/journals/predatory/list/hijacked");
  assert.equal(JOURNAL_PATHS.index("predatory_journal", "a"), "/journals/predatory/list/journals?letter=a");
  assert.equal(JOURNAL_PATHS.index("predatory_publisher", "b", 3), "/journals/predatory/list/publishers?letter=b&page=3");
  assert.equal(JOURNAL_PATHS.index("predatory_publisher", undefined, 1), "/journals/predatory/list/publishers");
});

test("list segments map back to list types", () => {
  assert.equal(listTypeFromSegment("journals"), "predatory_journal");
  assert.equal(listTypeFromSegment("publishers"), "predatory_publisher");
  assert.equal(listTypeFromSegment("hijacked"), "hijacked");
  assert.equal(listTypeFromSegment("users"), null);
});

test("letters and slugs are validated before reaching the backend", () => {
  assert.ok(isLetter("a") && isLetter("other"));
  assert.ok(!isLetter("ab") && !isLetter("A") && !isLetter(undefined));
  assert.ok(isEntrySlug("abc-journal-5762630ef0"));
  assert.ok(!isEntrySlug("../etc") && !isEntrySlug("") && !isEntrySlug("-x") && !isEntrySlug("Upper"));
});
