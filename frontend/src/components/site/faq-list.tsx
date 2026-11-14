import { ChevronDown } from "lucide-react";
import { Bidi } from "@/components/i18n/bidi";
import type { TextLang } from "@/lib/i18n/text-lang";
import { RichText } from "./rich-text";

export interface FaqListItem {
  question: string;
  answer: string;
  /** Set for site-document entries, whose language may differ from the page's. */
  questionLang?: TextLang;
  answerLang?: TextLang;
}

/** Accessible, script-free FAQ (native <details>); answers are plain text. */
export function FaqList({ items }: { items: FaqListItem[] }) {
  return (
    <div className="divide-y rounded-2xl border bg-card">
      {items.map((item, i) => (
        <details key={i} className="group p-0 [&_summary::-webkit-details-marker]:hidden" open={i === 0}>
          <summary className="flex cursor-pointer list-none items-center justify-between gap-4 px-5 py-4 font-semibold text-primary hover:text-accent focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none">
            {item.questionLang ? <Bidi text={item.question} lang={item.questionLang} /> : <span>{item.question}</span>}
            <ChevronDown className="size-5 shrink-0 transition-transform group-open:rotate-180 motion-reduce:transition-none" aria-hidden />
          </summary>
          <div className="px-5 pb-5">
            <RichText text={item.answer} lang={item.answerLang} className="text-muted-foreground" />
          </div>
        </details>
      ))}
    </div>
  );
}
