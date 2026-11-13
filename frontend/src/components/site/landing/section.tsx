import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

interface SectionProps {
  id: string;
  title: string;
  children: ReactNode;
  action?: ReactNode;
  intro?: string;
  className?: string;
}

/** A titled landing-page section; the title labels the region for assistive technology. */
export function Section({ id, title, children, action, intro, className }: SectionProps) {
  return (
    <section id={id} aria-labelledby={`${id}-title`} className={cn("mx-auto max-w-6xl scroll-mt-24 px-4 pt-14", className)}>
      <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div className="min-w-0">
          <h2 id={`${id}-title`} className="border-s-4 border-accent ps-3 text-xl font-bold text-primary md:text-2xl">
            {title}
          </h2>
          {intro && <p className="mt-2 ps-4 text-muted-foreground">{intro}</p>}
        </div>
        {action}
      </div>
      {children}
    </section>
  );
}
