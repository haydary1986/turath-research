"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { Plus, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { CO_RESEARCHER_KEYS, MAX_CO_RESEARCHERS } from "@/lib/portal/fields";

interface CoResearchersFieldProps {
  details: Record<string, string>;
  onChange: (key: string, value: string) => void;
}

/** Names of the partners of a joint plan: one to three rows, added and removed by the applicant. */
export function CoResearchersField({ details, onChange }: CoResearchersFieldProps) {
  const t = useTranslations("portal");
  const filled = CO_RESEARCHER_KEYS.filter((key) => details[key]?.trim()).length;
  const [rows, setRows] = useState(Math.max(1, filled));

  function remove(index: number) {
    // Shift the later names up so the stored keys stay contiguous (1, 2, 3).
    const names = CO_RESEARCHER_KEYS.map((key) => details[key] ?? "");
    const kept = [...names.slice(0, index), ...names.slice(index + 1), ""];
    CO_RESEARCHER_KEYS.forEach((key, i) => onChange(key, kept[i]));
    setRows((n) => Math.max(1, n - 1));
  }

  return (
    <fieldset className="space-y-3 rounded-lg border p-3 md:col-span-2">
      <legend className="px-1 text-sm font-medium">
        {t("fields.co_researchers")} *
      </legend>
      <p className="text-xs text-muted-foreground">{t("ui.coResearchersHint", { max: MAX_CO_RESEARCHERS })}</p>
      {CO_RESEARCHER_KEYS.slice(0, rows).map((key, index) => (
        <div key={key} className="flex items-end gap-2">
          <div className="flex-1 space-y-1">
            <Label htmlFor={`detail-${key}`} className="text-xs text-muted-foreground">
              {t("ui.coResearcherN", { n: index + 1 })}
            </Label>
            <Input
              id={`detail-${key}`}
              value={details[key] ?? ""}
              maxLength={255}
              onChange={(e) => onChange(key, e.target.value)}
            />
          </div>
          {rows > 1 && (
            <Button
              type="button"
              variant="ghost"
              size="icon"
              aria-label={t("ui.removeCoResearcher", { n: index + 1 })}
              onClick={() => remove(index)}
            >
              <X className="size-4" aria-hidden />
            </Button>
          )}
        </div>
      ))}
      {rows < MAX_CO_RESEARCHERS && (
        <Button type="button" variant="outline" size="sm" onClick={() => setRows((n) => n + 1)}>
          <Plus className="size-4" aria-hidden />
          {t("ui.addCoResearcher")}
        </Button>
      )}
    </fieldset>
  );
}
