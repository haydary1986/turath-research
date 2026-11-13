import { Fragment } from "react";
import { cn } from "@/lib/utils";
import { parseRichText, type Inline } from "@/lib/site/rich-text";

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

/** Renders editor text (paragraphs, bullet lists, https links) as React elements — never raw HTML. */
export function RichText({ text, className }: { text: string; className?: string }) {
  const blocks = parseRichText(text);
  return (
    <div className={cn("space-y-3 leading-loose", className)}>
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
