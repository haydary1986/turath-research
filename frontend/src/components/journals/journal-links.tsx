import { getTranslations } from "next-intl/server";
import { BookOpenText, SearchCheck, ShieldAlert, type LucideIcon } from "lucide-react";
import { Link } from "@/i18n/routing";
import { articlesFor, GUIDE_KEY, TOPIC_SLUGS } from "@/lib/journals/articles";
import { JOURNAL_PATHS } from "@/lib/journals/lists";

interface Card {
  href: string;
  title: string;
  body: string;
  icon: LucideIcon;
}

/** Cross-links between the guide hub, the topic articles and the tools (the current page is left out). */
export async function JournalLinks({ locale, current, heading }: { locale: string; current?: string; heading: string }) {
  const articles = articlesFor(locale);
  const t = await getTranslations({ locale, namespace: "journals" });
  const cards: Card[] = [
    { href: JOURNAL_PATHS.guide, title: articles[GUIDE_KEY].navLabel, body: articles[GUIDE_KEY].description, icon: ShieldAlert },
    ...TOPIC_SLUGS.map((slug) => ({
      href: JOURNAL_PATHS.topic(slug), title: articles[slug].navLabel, body: articles[slug].description, icon: BookOpenText,
    })),
    { href: JOURNAL_PATHS.check, title: t("crumbs.check"), body: t("checkPage.description"), icon: SearchCheck },
  ].filter((card) => card.href !== current);

  return (
    <section aria-labelledby="journal-links-title" className="mt-14">
      <h2 id="journal-links-title" className="mb-4 text-xl font-bold text-primary">{heading}</h2>
      <ul className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {cards.map(({ href, title, body, icon: Icon }) => (
          <li key={href}>
            <Link
              href={href}
              className="group flex h-full gap-3 rounded-xl border bg-card p-4 hover:border-accent/40 hover:shadow-sm focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
            >
              <Icon className="mt-0.5 size-5 shrink-0 text-accent" aria-hidden />
              <span className="min-w-0">
                <span className="block font-semibold text-primary group-hover:text-accent">{title}</span>
                <span className="mt-1 line-clamp-2 block text-sm text-muted-foreground">{body}</span>
              </span>
            </Link>
          </li>
        ))}
      </ul>
    </section>
  );
}
