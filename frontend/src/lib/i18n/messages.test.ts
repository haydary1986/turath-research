import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { test } from "node:test";
import { fileURLToPath } from "node:url";

/**
 * Every message key must exist in both locales, for both message sets (the portal's
 * and the public site's), and the two sets must not define the same top-level
 * namespace (they are merged with a shallow spread in src/i18n/request.ts).
 */

const MESSAGES = join(dirname(fileURLToPath(import.meta.url)), "../../messages");
const SETS = [
  { name: "portal", dir: MESSAGES },
  { name: "public", dir: join(MESSAGES, "public") },
];
const ARABIC = /[ء-ي]/;

type Tree = { [key: string]: Tree | string };

function load(dir: string, locale: string): Tree {
  return JSON.parse(readFileSync(join(dir, `${locale}.json`), "utf8")) as Tree;
}

function leaves(tree: Tree, prefix = ""): Map<string, string> {
  const out = new Map<string, string>();
  for (const [key, value] of Object.entries(tree)) {
    const path = prefix ? `${prefix}.${key}` : key;
    if (typeof value === "string") out.set(path, value);
    else for (const [k, v] of leaves(value, path)) out.set(k, v);
  }
  return out;
}

for (const set of SETS) {
  const ar = leaves(load(set.dir, "ar"));
  const en = leaves(load(set.dir, "en"));

  test(`${set.name} messages: every Arabic key exists in English`, () => {
    assert.deepEqual([...ar.keys()].filter((k) => !en.has(k)), []);
  });

  test(`${set.name} messages: every English key exists in Arabic`, () => {
    assert.deepEqual([...en.keys()].filter((k) => !ar.has(k)), []);
  });

  test(`${set.name} messages: no empty values`, () => {
    const empty = [...ar, ...en].filter(([, v]) => v.trim() === "").map(([k]) => k);
    assert.deepEqual(empty, []);
  });

  test(`${set.name} messages: English values contain no Arabic letters`, () => {
    // The language switcher label is the one deliberate exception: it names the other language.
    const allowed = new Set(["site.nav.switchLanguage"]);
    const arabic = [...en].filter(([k, v]) => !allowed.has(k) && ARABIC.test(v)).map(([k]) => k);
    assert.deepEqual(arabic, []);
  });
}

test("the two message sets do not share a top-level namespace", () => {
  for (const locale of ["ar", "en"]) {
    const portal = Object.keys(load(SETS[0].dir, locale));
    const site = new Set(Object.keys(load(SETS[1].dir, locale)));
    assert.deepEqual(portal.filter((k) => site.has(k)), [], locale);
  }
});
