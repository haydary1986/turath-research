import { getTranslations } from "next-intl/server";
import { CheckCircle2, Eye, Flag, ListChecks, Target } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { Localized } from "@/components/i18n/localized";
import type { SiteDepartment, SiteDeptItem } from "@/lib/site/types";
import { Section } from "./section";

function Statement({ icon: Icon, title, ar, en, locale }: {
  icon: LucideIcon; title: string; ar: string; en: string; locale: string;
}) {
  return (
    <div className="rounded-2xl border bg-card p-6">
      <h3 className="flex items-center gap-2 text-lg font-bold text-primary">
        <span className="flex size-9 items-center justify-center rounded-lg bg-primary text-primary-foreground">
          <Icon className="size-5" aria-hidden />
        </span>
        {title}
      </h3>
      <Localized as="p" block ar={ar} en={en} locale={locale} className="mt-3 leading-loose text-foreground/90" />
    </div>
  );
}

function ItemList({ icon: Icon, title, items, locale, columns }: {
  icon: LucideIcon; title: string; items: SiteDeptItem[]; locale: string; columns?: boolean;
}) {
  return (
    <div className="rounded-2xl border bg-card p-6">
      <h3 className="flex items-center gap-2 text-lg font-bold text-primary">
        <Icon className="size-5 text-accent" aria-hidden />
        {title}
      </h3>
      <ul className={columns ? "mt-4 grid gap-x-8 gap-y-3 md:grid-cols-2" : "mt-4 space-y-3"}>
        {items.map((item, i) => (
          <li key={i} className="flex gap-2.5 leading-relaxed">
            <CheckCircle2 className="mt-1 size-4 shrink-0 text-accent" aria-hidden />
            <Localized ar={item.text} en={item.text_en} locale={locale} />
          </li>
        ))}
      </ul>
    </div>
  );
}

/** The department's vision, mission, goals and tasks; each part hides itself when empty. */
export async function DepartmentProfile({ locale, department }: { locale: string; department: SiteDepartment }) {
  const { vision, vision_en, mission, mission_en, goals, tasks } = department;
  if (!vision && !mission && goals.length === 0 && tasks.length === 0) return null;
  const t = await getTranslations({ locale, namespace: "landing.department" });

  return (
    <Section id="department" title={t("title")}>
      <div className="space-y-4">
        {(vision || mission) && (
          <div className="grid gap-4 md:grid-cols-2">
            {vision && <Statement icon={Eye} title={t("vision")} ar={vision} en={vision_en} locale={locale} />}
            {mission && <Statement icon={Flag} title={t("mission")} ar={mission} en={mission_en} locale={locale} />}
          </div>
        )}
        {goals.length > 0 && <ItemList icon={Target} title={t("goals")} items={goals} locale={locale} columns />}
        {tasks.length > 0 && <ItemList icon={ListChecks} title={t("tasks")} items={tasks} locale={locale} columns />}
      </div>
    </Section>
  );
}
