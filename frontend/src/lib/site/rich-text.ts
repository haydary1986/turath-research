/**
 * A deliberately tiny "rich text" format for editor-written content: plain text where
 * a blank line starts a new paragraph, lines starting with "- " (or "• ", "* ") form a
 * bulleted list, and bare https:// URLs become links. The output is a data structure
 * rendered by React components — never HTML — so nothing an editor types can inject markup.
 */

export type Inline = { kind: "text"; text: string } | { kind: "link"; href: string; text: string };

export type Block =
  | { kind: "paragraph"; lines: Inline[][] }
  | { kind: "list"; items: Inline[][] };

const BULLET = /^\s*(?:[-*•])\s+/;
const URL_PATTERN = /https:\/\/[^\s<>"'`]+/g;
const TRAILING_PUNCTUATION = /[.,;:!?،؛)\]»"']+$/;

/** Splits a line into text and https-link segments. */
export function parseInline(line: string): Inline[] {
  const out: Inline[] = [];
  let last = 0;
  for (const match of line.matchAll(URL_PATTERN)) {
    const start = match.index ?? 0;
    const raw = match[0];
    const href = raw.replace(TRAILING_PUNCTUATION, "");
    if (!isLinkable(href)) continue;
    if (start > last) out.push({ kind: "text", text: line.slice(last, start) });
    out.push({ kind: "link", href, text: href });
    last = start + href.length;
  }
  if (last < line.length) out.push({ kind: "text", text: line.slice(last) });
  return out;
}

function isLinkable(href: string): boolean {
  try {
    const url = new URL(href);
    return url.protocol === "https:" && url.hostname !== "" && url.username === "" && url.password === "";
  } catch {
    return false;
  }
}

/** Parses editor text into paragraphs and bulleted lists. */
export function parseRichText(source: string): Block[] {
  const lines = source.replace(/\r\n?/g, "\n").split("\n");
  const blocks: Block[] = [];
  let current: Block | null = null;

  const flush = () => {
    if (current) blocks.push(current);
    current = null;
  };

  for (const rawLine of lines) {
    const line = rawLine.trimEnd();
    if (line.trim() === "") {
      flush();
      continue;
    }
    if (BULLET.test(line)) {
      const item = parseInline(line.replace(BULLET, ""));
      if (current?.kind !== "list") {
        flush();
        current = { kind: "list", items: [] };
      }
      current = { kind: "list", items: [...current.items, item] };
      continue;
    }
    const inline = parseInline(line.trim());
    if (current?.kind !== "paragraph") {
      flush();
      current = { kind: "paragraph", lines: [] };
    }
    current = { kind: "paragraph", lines: [...current.lines, inline] };
  }
  flush();
  return blocks;
}

/** First paragraph as plain text, cut to `max` characters — for cards and descriptions. */
export function excerpt(source: string, max: number): string {
  const text = source
    .split(/\n\s*\n/)
    .map((chunk) => chunk.replace(BULLET, "").replace(/\s+/g, " ").trim())
    .find((chunk) => chunk !== "") ?? "";
  if (text.length <= max) return text;
  return `${text.slice(0, max).trimEnd()}…`;
}
