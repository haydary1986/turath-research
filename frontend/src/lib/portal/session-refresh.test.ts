import assert from "node:assert/strict";
import { test } from "node:test";
import { SESSION_REFRESH_MIN_GAP_MS, refreshDue, shouldSignOut } from "./session-refresh.ts";

test("the stored user is refreshed on first mount, per sign-in, and after the gap", () => {
  assert.ok(refreshDue(null, "t1", 0, false));
  const last = { token: "t1", at: 1_000 };
  assert.ok(!refreshDue(last, "t1", 1_000 + SESSION_REFRESH_MIN_GAP_MS - 1, false), "focus storms do not refetch");
  assert.ok(refreshDue(last, "t1", 1_000 + SESSION_REFRESH_MIN_GAP_MS, false));
  assert.ok(refreshDue(last, "t2", 1_001, false), "a new sign-in loads at once");
  assert.ok(refreshDue(last, "t1", 1_001, true), "a forced reload ignores the gap");
});

test("only a 401 signs the user out", () => {
  assert.ok(shouldSignOut(401));
  assert.ok(!shouldSignOut(403));
  assert.ok(!shouldSignOut(503), "a backend hiccup keeps the session");
  assert.ok(!shouldSignOut(undefined), "a network failure keeps the session");
});
