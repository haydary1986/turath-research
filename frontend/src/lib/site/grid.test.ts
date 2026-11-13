import assert from "node:assert/strict";
import { test } from "node:test";
import { balancedCardClass } from "./grid.ts";

/** Splits `count` cards into rows of a grid with `tracks` tracks, given each card's span. */
function rows(spans: number[], tracks: number): number[] {
  const out: number[] = [];
  let used = tracks;
  for (const span of spans) {
    if (used + span > tracks) {
      out.push(0);
      used = 0;
    }
    out[out.length - 1] += 1;
    used += span;
  }
  return out;
}

function spansAt(count: number, prefix: "sm" | "lg", base: number): number[] {
  return Array.from({ length: count }, (_, i) => {
    const match = balancedCardClass(i, count).match(new RegExp(`${prefix}:col-span-(\\d)`));
    return match ? Number(match[1]) : base;
  });
}

test("no breakpoint leaves a lone card on a row with free space", () => {
  for (let count = 2; count <= 12; count++) {
    const sm = spansAt(count, "sm", 1);
    let start = 0;
    rows(sm, 2).forEach((n, r) => {
      const width = sm.slice(start, start + n).reduce((a, b) => a + b, 0);
      assert.equal(width, 2, `sm row ${r} of ${count} cards is not full`);
      start += n;
    });

    const lg = spansAt(count, "lg", 2);
    const lgRows = rows(lg, 6);
    start = 0;
    lgRows.forEach((n, r) => {
      const width = lg.slice(start, start + n).reduce((a, b) => a + b, 0);
      assert.equal(width, 6, `lg row ${r} of ${count} cards is not full`);
      assert.ok(n >= 2, `lg row ${r} of ${count} cards has a single card`);
      start += n;
    });
  }
});

test("a single card spans the whole row", () => {
  assert.match(balancedCardClass(0, 1), /lg:col-span-6/);
});
