import assert from "node:assert/strict";
import { test } from "node:test";
import {
  EMPTY_FILTERS, activeFilterCount, filtersFromParams, filtersToApiQuery, filtersToParams, withFilter,
} from "./filters.ts";

test("reads valid filters and drops malformed ones", () => {
  const f = filtersFromParams(new URLSearchParams("q=نانو&year_from=2020&year_to=20x&type=poem&sort=bad&page=-2&author=علي"));
  assert.equal(f.q, "نانو");
  assert.equal(f.author, "علي");
  assert.equal(f.year_from, "2020");
  assert.equal(f.year_to, "");
  assert.equal(f.type, "");
  assert.equal(f.sort, "newest");
  assert.equal(f.page, 1);
});

test("the URL keeps only non-default values", () => {
  assert.equal(filtersToParams(EMPTY_FILTERS).toString(), "");
  const params = filtersToParams({ ...EMPTY_FILTERS, type: "book", sort: "cited", page: 2 });
  assert.equal(params.toString(), "type=book&sort=cited&page=2");
});

test("the API query always carries sort, page and size", () => {
  assert.equal(filtersToApiQuery(EMPTY_FILTERS), "sort=newest&page=1&limit=20");
});

test("changing a filter returns to page one and a new college clears the department", () => {
  const start = { ...EMPTY_FILTERS, college_id: "c1", department_id: "d1", page: 4 };
  const next = withFilter(start, { college_id: "c2" });
  assert.equal(next.page, 1);
  assert.equal(next.department_id, "");
  assert.equal(withFilter(start, { q: "x" }).department_id, "d1");
});

test("counts active filters", () => {
  assert.equal(activeFilterCount({ ...EMPTY_FILTERS, q: "x", year_from: "2020", sort: "cited" }), 2);
});
