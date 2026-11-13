import assert from "node:assert/strict";
import { test } from "node:test";
import {
  ACTIVITY_CATEGORIES,
  canExportFS003,
  enterableCategories,
  fs003Filename,
  holds,
  isDetailRequired,
  isInstitutional,
  isPlaceCategory,
  MAX_APPROVAL_NO_LENGTH,
  MAX_DECISION_NOTE_LENGTH,
  activityStatusKey,
  currentDecisionNote,
  validateApproval,
  validateRefusal,
} from "./activity.ts";

test("only the owner of a returned activity is addressed directly", () => {
  assert.equal(activityStatusKey("returned", true), "returnedToYou");
  assert.equal(activityStatusKey("returned", false), "returned");
  assert.equal(activityStatusKey("pending", true), "pending");
});

test("a pending activity hides the note left from its earlier return", () => {
  assert.equal(currentDecisionNote("pending", "old note"), null);
  assert.equal(currentDecisionNote("returned", "fix it"), "fix it");
  assert.equal(currentDecisionNote("approved", null), null);
});

test("approval needs a non-blank number within the limit", () => {
  assert.deepEqual(validateApproval("   ", ""), ["approval_no:required"]);
  assert.deepEqual(validateApproval("77/2026", ""), []);
  assert.deepEqual(validateApproval("١".repeat(MAX_APPROVAL_NO_LENGTH), ""), []);
  assert.deepEqual(validateApproval("١".repeat(MAX_APPROVAL_NO_LENGTH + 1), ""), ["approval_no:too_long"]);
});

test("approval note is optional but bounded", () => {
  assert.deepEqual(validateApproval("1", "x".repeat(MAX_DECISION_NOTE_LENGTH + 1)), ["approval_note:too_long"]);
});

test("returning or rejecting needs a non-blank note within the limit", () => {
  assert.deepEqual(validateRefusal(" \n "), ["note:required"]);
  assert.deepEqual(validateRefusal("سبب واضح"), []);
  assert.deepEqual(validateRefusal("x".repeat(MAX_DECISION_NOTE_LENGTH + 1)), ["note:too_long"]);
});

test("FS 003 has thirteen categories and three institutional ones", () => {
  assert.equal(ACTIVITY_CATEGORIES.length, 13);
  assert.deepEqual(ACTIVITY_CATEGORIES.filter(isInstitutional), [
    "infrastructure", "international_agreements", "local_agreements",
  ]);
  assert.ok(isPlaceCategory("workshops") && !isPlaceCategory("books"));
  assert.ok(isDetailRequired("books") && !isDetailRequired("workshops") && !isDetailRequired("infrastructure"));
});

test("responsibility scope: university-wide covers every college, a college scope only its own", () => {
  const scopes = [
    { category: "international_agreements", college_id: null },
    { category: "local_agreements", college_id: "c1" },
  ];
  assert.ok(holds(scopes, "international_agreements", null));
  assert.ok(holds(scopes, "international_agreements", "c9"));
  assert.ok(holds(scopes, "local_agreements", "c1"));
  assert.ok(!holds(scopes, "local_agreements", "c2"));
  assert.ok(!holds(scopes, "local_agreements", null));
  assert.ok(!holds(scopes, "books", "c1"));
});

test("researchers cannot pick institutional categories without a responsibility", () => {
  const plain = enterableCategories("researcher", []);
  assert.ok(plain.includes("workshops") && plain.includes("books"));
  assert.ok(!plain.includes("infrastructure") && !plain.includes("local_agreements"));
  const holder = enterableCategories("researcher", [{ category: "local_agreements", college_id: "c1" }]);
  assert.ok(holder.includes("local_agreements") && !holder.includes("international_agreements"));
  assert.equal(enterableCategories("scientific_affairs", []).length, 13);
  assert.ok(!enterableCategories("vp_research", []).includes("infrastructure"));
});

test("FS 003 export is for staff and responsibility holders", () => {
  assert.ok(!canExportFS003("researcher", []));
  assert.ok(canExportFS003("researcher", [{ category: "books", college_id: null }]));
  assert.ok(canExportFS003("vp_research", []));
  assert.equal(fs003Filename("2026-11-01", "2026-11-30"), "fs003-2026-11-01_2026-11-30.xlsx");
});

test("the date column is named per sheet", async () => {
  const { dateLabelKey } = await import("./activity.ts");
  assert.equal(dateLabelKey("indexed_research"), "research");
  assert.equal(dateLabelKey("local_agreements"), "agreements");
  assert.equal(dateLabelKey("patents"), "patents");
  assert.equal(dateLabelKey("workshops"), "default");
});
