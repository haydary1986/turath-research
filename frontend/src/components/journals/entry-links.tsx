import { Link } from "@/i18n/routing";
import { JOURNAL_PATHS, type JournalEntryLink, type ListType } from "@/lib/journals/lists";
import { Bidi } from "@/components/i18n/bidi";

interface EntryLinksProps {
  id: string;
  title: string;
  links: JournalEntryLink[];
  listLabel: (type: ListType) => string;
}

/** Related entries (same domain, alphabetical neighbours) as plain crawlable links. */
export function EntryLinks({ id, title, links, listLabel }: EntryLinksProps) {
  if (links.length === 0) return null;
  return (
    <section aria-labelledby={id} className="mt-10">
      <h2 id={id} className="mb-3 text-lg font-bold text-primary">{title}</h2>
      <ul className="grid gap-2 sm:grid-cols-2">
        {links.map((link) => (
          <li key={link.slug}>
            <Link
              href={JOURNAL_PATHS.entry(link.slug)}
              className="flex h-full flex-col rounded-lg border bg-card px-3 py-2 hover:border-accent/40 focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
            >
              <Bidi text={link.name} className="font-medium text-primary" />
              <span className="text-xs text-muted-foreground">{listLabel(link.list_type)}</span>
            </Link>
          </li>
        ))}
      </ul>
    </section>
  );
}
