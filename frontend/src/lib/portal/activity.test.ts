import assert from "node:assert/strict";
import { test } from "node:test";
import {
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
