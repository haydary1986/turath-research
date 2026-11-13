import assert from "node:assert/strict";
import { test } from "node:test";
import { isSafeServiceLink, requiresSignIn } from "./types.ts";

test("service links stay inside the site's portal, journal and news pages", () => {
  for (const ok of ["/portal", "/portal/new/research_plan", "/journals/check", "/news"]) {
    assert.ok(isSafeServiceLink(ok), ok);
  }
  for (const bad of ["", "https://evil.iq", "//evil.iq", "/admin", "/portal/../admin", "/portalx", "javascript:alert(1)", "/Portal"]) {
    assert.ok(!isSafeServiceLink(bad), bad);
  }
});

test("only portal pages need a session", () => {
  assert.ok(requiresSignIn("/portal/activities"));
  assert.ok(!requiresSignIn("/journals/check"));
});
