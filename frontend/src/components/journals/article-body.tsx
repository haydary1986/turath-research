import type { Article } from "@/lib/journals/article-types";

function sectionId(index: number): string {
  return `section-${index + 1}`;
}

/** The headed sections of a guide article, with an in-page table of contents. */
export function ArticleBody({ article, tocLabel }: { article: Article; tocLabel: string }) {
  return (
    <div className="space-y-8">
      {article.sections.length > 2 && (
        <nav aria-label={tocLabel} className="rounded-xl border bg-muted/40 p-4">
          <p className="mb-2 text-sm font-semibold text-primary">{tocLabel}</p>
          <ol className="list-decimal space-y-1 ps-5 text-sm">
            {article.sections.map((section, i) => (
              <li key={section.heading}>
                <a href={`#${sectionId(i)}`} className="text-primary hover:text-accent hover:underline">{section.heading}</a>
              </li>
            ))}
          </ol>
        </nav>
      )}
      {article.sections.map((section, i) => (
        <section key={section.heading} aria-labelledby={sectionId(i)} className="scroll-mt-24">
          <h2 id={sectionId(i)} className="text-xl font-bold text-primary md:text-2xl">{section.heading}</h2>
          {section.paragraphs.map((p, j) => (
            <p key={j} className="mt-3 leading-loose text-foreground/90">{p}</p>
          ))}
          {section.bullets && (
            <ul className="mt-3 space-y-2">
              {section.bullets.map((b, j) => (
                <li key={j} className="flex gap-2.5 leading-relaxed">
                  <span className="mt-2.5 size-1.5 shrink-0 rounded-full bg-accent" aria-hidden />
                  <span>{b}</span>
                </li>
              ))}
            </ul>
          )}
        </section>
      ))}
    </div>
  );
}
