import assert from "node:assert/strict";
import { test } from "node:test";
import { serializeJsonLd } from "./json-ld.ts";
import { SITE_URL, absoluteUrl, metaDescription, pageAlternates } from "./site.ts";

test("absolute URLs are locale-prefixed and the locale root has no trailing slash", () => {
  assert.equal(absoluteUrl("ar", "/"), `${SITE_URL}/ar`);
  assert.equal(absoluteUrl("en", "/journals/check"), `${SITE_URL}/en/journals/check`);
  assert.equal(absoluteUrl("fr", "/x"), `${SITE_URL}/ar/x`);
});

test("alternates point at both languages with Arabic as x-default", () => {
  const alt = pageAlternates("en", "/journals");
  assert.equal(alt.canonical, `${SITE_URL}/en/journals`);
  assert.deepEqual(alt.languages, {
    ar: `${SITE_URL}/ar/journals`,
    en: `${SITE_URL}/en/journals`,
    "x-default": `${SITE_URL}/ar/journals`,
  });
});

test("meta descriptions are trimmed on a word boundary", () => {
  const long = "word ".repeat(60);
  const out = metaDescription(long, 50);
  assert.ok(out.length <= 50);
  assert.ok(out.endsWith("…"));
  assert.equal(metaDescription("  short   text "), "short text");
});

test("JSON-LD cannot close its script tag", () => {
  const out = serializeJsonLd({ name: "</script><script>alert(1)</script>" });
  assert.ok(!out.includes("</script>"));
  assert.deepEqual(JSON.parse(out), { name: "</script><script>alert(1)</script>" });
});
