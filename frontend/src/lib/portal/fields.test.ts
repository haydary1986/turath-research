import assert from "node:assert/strict";
import { test } from "node:test";
import { coResearchers, fieldsFor, hasJournalData, withWorkType } from "./fields.ts";

test("coResearchers lists non-blank names in order", () => {
  assert.deepEqual(coResearchers({ co_researcher_1: " أ ", co_researcher_2: "", co_researcher_3: "ج" }), ["أ", "ج"]);
});

test("an individual plan drops its partners", () => {
  const next = withWorkType({ work_type: "joint", co_researcher_1: "أ", notes: "x" }, "individual");
  assert.deepEqual(next, { work_type: "individual", notes: "x" });
});

test("a joint plan keeps its partners", () => {
  assert.deepEqual(withWorkType({ co_researcher_1: "أ" }, "joint"), { co_researcher_1: "أ", work_type: "joint" });
});

test("a title change asks for the old title and the reason only", () => {
  assert.deepEqual(fieldsFor("title_change").map((f) => f.key), ["old_title", "reason"]);
  assert.equal(hasJournalData("title_change"), false);
  assert.equal(hasJournalData("research_plan"), false);
  assert.equal(hasJournalData("appreciation_letter"), true);
});
