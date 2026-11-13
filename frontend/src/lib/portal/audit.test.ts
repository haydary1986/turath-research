import assert from "node:assert/strict";
import { test } from "node:test";
import { formatBaghdadTime, loginLogQuery } from "./audit.ts";

test("login log query leaves out empty filters", () => {
  const q = loginLogQuery({ q: " ahmed ", method: "", result: "failure", from: "2026-11-01", to: "" }, 2, 25);
  assert.equal(q, "page=2&limit=25&q=ahmed&result=failure&from=2026-11-01");
});

test("timestamps are shown on Baghdad time", () => {
  const shown = formatBaghdadTime("2026-11-13T21:30:00Z", "en");
  assert.match(shown, /14\/11\/2026/);
  assert.match(shown, /00:30/);
  assert.equal(formatBaghdadTime(null, "en"), "");
  assert.equal(formatBaghdadTime("garbage", "en"), "");
});
