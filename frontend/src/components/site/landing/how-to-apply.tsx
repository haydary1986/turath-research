import { getTranslations } from "next-intl/server";
import { FileUp, LogIn, Route, type LucideIcon } from "lucide-react";
import { formatNumber } from "@/lib/site/format";
import { Section } from "./section";

const STEPS: { icon: LucideIcon; key: "step1" | "step2" | "step3" }[] = [
  { icon: LogIn, key: "step1" },
  { icon: FileUp, key: "step2" },
  { icon: Route, key: "step3" },
];

/** «كيف أقدم طلبًا؟» — three illustrated steps. */
export async function HowToApply({ locale }: { locale: string }) {
  const t = await getTranslations({ locale, namespace: "landing.howTo" });
  return (
    <Section id="how-to-apply" title={t("title")} intro={t("intro")}>
      <ol className="grid gap-4 md:grid-cols-3">
        {STEPS.map(({ icon: Icon, key }, i) => (
          <li key={key} className="relative flex gap-4 rounded-2xl border bg-card p-5 md:flex-col">
            <div className="relative shrink-0">
              <span className="flex size-14 items-center justify-center rounded-2xl bg-primary/5 text-primary ring-1 ring-primary/10">
                <Icon className="size-7" aria-hidden />
              </span>
              <span
                className="absolute -end-2 -top-2 flex size-7 items-center justify-center rounded-full bg-accent text-sm font-bold text-accent-foreground tabular-nums"
                aria-hidden
              >
                {formatNumber(i + 1, locale)}
              </span>
            </div>
            <div className="min-w-0">
              <p className="sr-only">{t("stepN", { n: i + 1 })}</p>
              <h3 className="font-bold text-primary">{t(`${key}Title`)}</h3>
              <p className="mt-1.5 text-sm leading-relaxed text-muted-foreground">{t(`${key}Body`)}</p>
            </div>
          </li>
        ))}
      </ol>
    </Section>
  );
}
