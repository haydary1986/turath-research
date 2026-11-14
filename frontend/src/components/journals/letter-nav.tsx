import { Link } from "@/i18n/routing";
import { JOURNAL_PATHS, LETTER_OTHER, type JournalLetter, type ListType } from "@/lib/journals/lists";
import { cn } from "@/lib/utils";

interface LetterNavProps {
  type: ListType;
  letters: JournalLetter[];
  label: string;
  allLabel: string;
  otherLabel: string;
  current?: string;
}

/** Crawlable A–Z links for one list (only letters that have entries). */
export function LetterNav({ type, letters, label, allLabel, otherLabel, current }: LetterNavProps) {
  const chip =
    "inline-flex h-9 min-w-9 items-center justify-center rounded-lg border px-2 text-sm font-semibold uppercase hover:border-accent hover:text-accent focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none aria-[current=page]:border-primary aria-[current=page]:bg-primary aria-[current=page]:text-primary-foreground";
  return (
    <nav aria-label={label}>
      <ul className="flex flex-wrap gap-1.5" dir="ltr">
        <li>
          <Link href={JOURNAL_PATHS.index(type)} aria-current={current ? undefined : "page"} className={cn(chip, "normal-case")}>
            {allLabel}
          </Link>
        </li>
        {letters.map(({ letter, count }) => (
          <li key={letter}>
            <Link
              href={JOURNAL_PATHS.index(type, letter)}
              aria-current={current === letter ? "page" : undefined}
              title={String(count)}
              className={chip}
              // The A–Z letters index Latin journal names: data, not UI text.
              data-bidi={letter === LETTER_OTHER ? undefined : ""}
            >
              {letter === LETTER_OTHER ? otherLabel : letter}
            </Link>
          </li>
        ))}
      </ul>
    </nav>
  );
}
