import assert from "node:assert/strict";
import { test } from "node:test";
import { contactEntries, needsIdentityReview } from "./research-profile.ts";

test("only name-check refusals ask for staff review", () => {
  for (const code of ["orcid_name_mismatch", "orcid_name_unverified", "openalex_name_mismatch"]) {
    assert.equal(needsIdentityReview(code), true);
  }
  for (const code of ["orcid_unavailable", "openalex_truncated", "sync_busy", "", null, undefined]) {
    assert.equal(needsIdentityReview(code), false);
  }
});

test("the contact card lists filled values only, with the right link kind", () => {
  const entries = contactEntries({
    email: "a@test.iq",
    phone: "07701234567",
    phone_secondary: null,
    recovery_email: "b@test.iq",
    employee_no: "",
    account_status: "linked",
  });
  assert.deepEqual(
    entries.map(({ key, kind }) => [key, kind]),
    [["contactEmail", "mailto"], ["phone", "tel"], ["recovery_email", "mailto"]],
  );
});
