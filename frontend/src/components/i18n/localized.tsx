import type { ReactNode } from "react";
import { pickLocalized } from "@/lib/i18n/text-lang";
import { Bidi } from "./bidi";

type LocalizedTag = "span" | "div" | "p" | "h1" | "h2" | "h3" | "dd" | "dt" | "strong" | "figcaption";

interface LocalizedProps {
  /** Arabic value (primary). */
  ar: string | null | undefined;
  /** English value (optional). */
  en: string | null | undefined;
  locale: string;
  as?: LocalizedTag;
  block?: boolean;
  className?: string;
  /** Rendered when both values are empty. */
  fallback?: ReactNode;
}

/**
 * Renders one bilingual field of the site document in the page's language, falling
 * back to Arabic when the English value is empty. The text always carries its own
 * `lang` and `dir`, so an Arabic fallback on an English page lays out right-to-left.
 */
export function Localized({ ar, en, locale, as, block, className, fallback = null }: LocalizedProps) {
  const picked = pickLocalized(ar, en, locale);
  if (!picked) return <>{fallback}</>;
  return <Bidi text={picked.text} lang={picked.lang} as={as} block={block} className={className} />;
}
