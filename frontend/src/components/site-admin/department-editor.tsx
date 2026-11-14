"use client";

import { useTranslations } from "next-intl";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { SITE_LIMITS, type SiteDepartment, type SiteDeptItem } from "@/lib/site/types";
import { BilingualField } from "./bilingual-field";
import { ListHeader, RowControls, move, removeAt, replaceAt } from "./editor-parts";

interface DepartmentEditorProps {
  department: SiteDepartment;
  onChange: (department: SiteDepartment) => void;
}

function ItemsEditor({ id, title, addLabel, itemLabel, items, onChange }: {
  id: string; title: string; addLabel: string; itemLabel: string;
  items: SiteDeptItem[]; onChange: (items: SiteDeptItem[]) => void;
}) {
  const tc = useTranslations("portal.site.content");
  return (
    <Card>
      <ListHeader title={title} addLabel={addLabel} canAdd={items.length < SITE_LIMITS.deptItems}
        onAdd={() => onChange([...items, { text: "", text_en: "" }])} />
      <CardContent className="space-y-3">
        {items.map((item, i) => (
          <fieldset key={i} className="flex items-start gap-2 rounded-lg border p-3">
            <legend className="sr-only">{tc("itemN", { n: i + 1 })}</legend>
            <BilingualField id={`${id}_${i}`} label={`${itemLabel} ${i + 1}`} required multiline rows={2}
              max={SITE_LIMITS.deptItem} className="flex-1" ar={item.text} en={item.text_en}
              onChange={(v) => onChange(replaceAt(items, i, { text: v.ar ?? item.text, text_en: v.en ?? item.text_en }))} />
            <RowControls index={i} count={items.length}
              onMove={(to) => onChange(move(items, i, to))} onRemove={() => onChange(removeAt(items, i))} />
          </fieldset>
        ))}
      </CardContent>
    </Card>
  );
}

/** Edits the department's vision, mission, goals and tasks shown on the homepage. */
export function DepartmentEditor({ department, onChange }: DepartmentEditorProps) {
  const t = useTranslations("siteEditor.department");
  const set = (patch: Partial<SiteDepartment>) => onChange({ ...department, ...patch });

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>{t("title")}</CardTitle>
          <p className="text-sm text-muted-foreground">{t("hint")}</p>
        </CardHeader>
        <CardContent className="space-y-4">
          <BilingualField id="dept_vision" label={t("vision")} multiline rows={3} max={SITE_LIMITS.deptStatement}
            ar={department.vision} en={department.vision_en}
            onChange={(v) => set({ vision: v.ar ?? department.vision, vision_en: v.en ?? department.vision_en })} />
          <BilingualField id="dept_mission" label={t("mission")} multiline rows={4} max={SITE_LIMITS.deptStatement}
            ar={department.mission} en={department.mission_en}
            onChange={(v) => set({ mission: v.ar ?? department.mission, mission_en: v.en ?? department.mission_en })} />
        </CardContent>
      </Card>
      <ItemsEditor id="dept_goal" title={t("goals")} addLabel={t("addGoal")} itemLabel={t("goal")}
        items={department.goals} onChange={(goals) => set({ goals })} />
      <ItemsEditor id="dept_task" title={t("tasks")} addLabel={t("addTask")} itemLabel={t("task")}
        items={department.tasks} onChange={(tasks) => set({ tasks })} />
    </div>
  );
}
