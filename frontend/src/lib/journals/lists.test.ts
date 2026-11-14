import assert from "node:assert/strict";
import { test } from "node:test";
import { JOURNAL_PATHS } from "./lists.ts";

test("the predatory-journals path stays stable for links already shared and indexed", () => {
  assert.equal(JOURNAL_PATHS.guide, "/journals/predatory");
});
