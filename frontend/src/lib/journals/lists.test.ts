import assert from "node:assert/strict";
import { test } from "node:test";
import { JOURNAL_PATHS } from "./lists.ts";

test("journal paths stay stable for links already shared and indexed", () => {
  assert.equal(JOURNAL_PATHS.guide, "/journals/predatory");
  assert.equal(JOURNAL_PATHS.check, "/journals/check");
  assert.equal(JOURNAL_PATHS.topic("signs"), "/journals/guide/signs");
});
