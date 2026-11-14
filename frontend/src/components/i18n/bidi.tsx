import type { ElementType, ReactNode } from "react";
import { cn } from "@/lib/utils";
import { detectLang, dirOf, type TextLang } from "@/lib/i18n/text-lang";

type BidiTag = "span" | "div" | "p" | "h1" | "h2" | "h3" | "h4" | "li" | "dd" | "dt" | "strong" | "figcaption" | "blockquote";

interface BidiProps {
  /** The data text (a name, title, note...). Empty text renders `fallback` instead. */
  text: string | null | undefined;
  /** The language the text is known to be in; detected from its letters when omitted. */
  lang?: TextLang | null;
  /** Render as a block (aligned to the text's own direction) instead of an inline isolate. */
  block?: boolean;
  as?: BidiTag;
  className?: string;
  title?: string;
  id?: string;
  /** Shown when the text is empty (e.g. "—"). */
  fallback?: ReactNode;
}

/**
 * Renders user-entered or database text so it lays out by its own script, whatever
 * the page language: it carries `lang` and `dir` (which isolates it from the
 * surrounding text), and `data-bidi` marks it as data for the i18n checks.
 * Text without letters (numbers, codes) gets `dir="auto"` and no `lang`.
 */
export function Bidi({ text, lang, block = false, as, className, title, id, fallback = null }: BidiProps) {
  if (!text) return <>{fallback}</>;
  const resolved = lang ?? detectLang(text);
  const Tag: ElementType = as ?? (block ? "div" : "span");
  return (
    <Tag
      id={id}
      lang={resolved ?? undefined}
      dir={resolved ? dirOf(resolved) : "auto"}
      data-bidi=""
      title={title}
      className={cn(block && "text-start", className)}
    >
      {text}
    </Tag>
  );
}

interface LtrProps {
  children: ReactNode;
  className?: string;
  title?: string;
}

/**
 * A left-to-right isolate for numbers, dates, DOIs, ISSNs, emails, URLs and codes,
 * so they keep their order inside Arabic text.
 */
export function Ltr({ children, className, title }: LtrProps) {
  if (children === null || children === undefined || children === "") return null;
  return (
    <span dir="ltr" data-bidi="" title={title} className={className}>
      {children}
    </span>
  );
}

/** Props to spread on an element whose whole content is data text of a known or detected language. */
export function bidiProps(text: string | null | undefined, lang?: TextLang | null): {
  lang?: string;
  dir: "rtl" | "ltr" | "auto";
  "data-bidi": "";
} {
  const resolved = lang ?? detectLang(text);
  return { lang: resolved ?? undefined, dir: resolved ? dirOf(resolved) : "auto", "data-bidi": "" };
}
