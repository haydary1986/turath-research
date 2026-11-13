import assert from "node:assert/strict";
import { test } from "node:test";
import { excerpt, parseInline, parseRichText } from "./rich-text.ts";

test("blank lines separate paragraphs and single newlines stay within one", () => {
  const blocks = parseRichText("سطر أول\nسطر ثان\n\n\nفقرة ثانية");
  assert.equal(blocks.length, 2);
  assert.deepEqual(blocks[0], {
    kind: "paragraph",
    lines: [[{ kind: "text", text: "سطر أول" }], [{ kind: "text", text: "سطر ثان" }]],
  });
});

test("bullet lines become a list", () => {
  const blocks = parseRichText("مقدمة\n- أولًا\n• ثانيًا\n* ثالثًا\nخاتمة");
  assert.deepEqual(blocks.map((b) => b.kind), ["paragraph", "list", "paragraph"]);
  const list = blocks[1];
  assert.equal(list.kind === "list" && list.items.length, 3);
});

test("only https URLs become links and trailing punctuation stays text", () => {
  assert.deepEqual(parseInline("راجع https://uoturath.edu.iq/x. ثم"), [
    { kind: "text", text: "راجع " },
    { kind: "link", href: "https://uoturath.edu.iq/x", text: "https://uoturath.edu.iq/x" },
    { kind: "text", text: ". ثم" },
  ]);
  assert.deepEqual(parseInline("http://a.iq javascript:alert(1)"), [
    { kind: "text", text: "http://a.iq javascript:alert(1)" },
  ]);
});

test("markup is kept as literal text", () => {
  const blocks = parseRichText("<script>alert(1)</script>");
  assert.deepEqual(blocks, [
    { kind: "paragraph", lines: [[{ kind: "text", text: "<script>alert(1)</script>" }]] },
  ]);
});

test("URLs with credentials are not linked", () => {
  assert.equal(parseInline("https://u:p@evil.example")[0].kind, "text");
});

test("excerpt takes the first paragraph and cuts it", () => {
  assert.equal(excerpt("\n\nأول فقرة طويلة\nتكملة\n\nثانية", 100), "أول فقرة طويلة تكملة");
  assert.equal(excerpt("abcdefghij", 4), "abcd…");
  assert.equal(excerpt("", 4), "");
});
