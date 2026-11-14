import { Fragment } from "react";
import { cn } from "@/lib/utils";
import { parseRichText, type Inline } from "@/lib/site/rich-text";
import { detectLang, dirOf, type TextLang } from "@/lib/i18n/text-lang";

function InlineLine({ parts }: { parts: Inline[] }) {
  return (
    <>
      {parts.map((part, i) =>
        part.kind === "link" ? (
          <a
            key={i}
            href={part.href}
            target="_blank"
            rel="noopener noreferrer nofollow"
            dir="ltr"
            className="break-all text-primary underline underline-offset-4 hover:text-accent"
          >
            {part.text}
          </a>
        ) : (
          <Fragment key={i}>{part.text}</Fragment>
        )
      )}
    </>
  );
}

interface RichTextProps {
  text: string;
  /** The language the text is written in; detected from its letters when omitted. */
  lang?: TextLang | null;
  className?: string;
}

/**
 * Renders editor text (paragraphs, bullet lists, https links) as React elements —
 * never raw HTML. The block carries the text's own lang/dir, so Arabic text on an
 * English page (or the reverse) is laid out in its own direction.
 */
export function RichText({ text, lang, className }: RichTextProps) {
  const blocks = parseRichText(text);
  const resolved = lang ?? detectLang(text);
  return (
    <div
      lang={resolved ?? undefined}
      dir={resolved ? dirOf(resolved) : undefined}
      data-bidi=""
      className={cn("space-y-3 text-start leading-loose", className)}
    >
      {blocks.map((block, i) =>
        block.kind === "list" ? (
          <ul key={i} className="list-disc space-y-1 ps-6 marker:text-accent">
            {block.items.map((item, j) => (
              <li key={j}>
                <InlineLine parts={item} />
              </li>
            ))}
          </ul>
        ) : (
          <p key={i}>
            {block.lines.map((line, j) => (
              <Fragment key={j}>
                {j > 0 && <br />}
                <InlineLine parts={line} />
              </Fragment>
            ))}
          </p>
        )
      )}
    </div>
  );
}
