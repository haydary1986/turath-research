"use client";

import { useTranslations } from "next-intl";
import { AlertTriangle, Info, ListChecks, Unlink } from "lucide-react";
import type { LucideIcon } from "lucide-react";

const SIGNS = ["sign1", "sign2", "sign3", "sign4", "sign5", "sign6"] as const;
const STEPS = ["step1", "step2", "step3", "step4"] as const;

function Block({ id, icon: Icon, title, children }: { id: string; icon: LucideIcon; title: string; children: React.ReactNode }) {
  return (
    <section aria-labelledby={id} className="rounded-xl border bg-card p-5">
      <h2 id={id} className="flex items-center gap-2 text-lg font-bold text-primary">
        <Icon className="size-5 shrink-0 text-accent" aria-hidden />
        {title}
      </h2>
      <div className="mt-3 text-sm leading-relaxed text-foreground/90">{children}</div>
    </section>
  );
}

/** A short explanation of predatory and hijacked journals and what to do before submitting. */
export function PredatoryExplainer() {
  const t = useTranslations("journals.explain");

  return (
    <div className="grid gap-4 md:grid-cols-2">
      <Block id="explain-what" icon={Info} title={t("whatTitle")}>
        <p>{t("what")}</p>
      </Block>
      <Block id="explain-hijacked" icon={Unlink} title={t("hijackedTitle")}>
        <p>{t("hijacked")}</p>
      </Block>
      <Block id="explain-signs" icon={AlertTriangle} title={t("signsTitle")}>
        <ul className="list-disc space-y-1.5 ps-5">
          {SIGNS.map((key) => <li key={key}>{t(key)}</li>)}
        </ul>
      </Block>
      <Block id="explain-steps" icon={ListChecks} title={t("stepsTitle")}>
        <ol className="list-decimal space-y-1.5 ps-5">
          {STEPS.map((key) => <li key={key}>{t(key)}</li>)}
        </ol>
      </Block>
    </div>
  );
}
