"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { api } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { bidiProps } from "@/components/i18n/bidi";
import { requestTitle, type PortalRequest } from "@/lib/portal/types";
import { NativeSelect } from "./form-controls";

interface TitleChangeSourceProps {
  value: string;
  /** Called with the chosen plan, or null when the research is not registered in the portal. */
  onSelect: (plan: PortalRequest | null) => void;
}

/** Old title of a registered plan formatted for the request: Arabic and English joined. */
export function planOldTitle(plan: PortalRequest): string {
  return [plan.title_ar, plan.title_en].filter((title) => title?.trim()).join("\n");
}

/** Picks the approved research plan whose title the applicant wants changed. */
export function TitleChangeSource({ value, onSelect }: TitleChangeSourceProps) {
  const t = useTranslations("portal");
  const [plans, setPlans] = useState<PortalRequest[]>([]);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    let active = true;
    api
      .get<PortalRequest[]>("/api/v1/portal/requests?mine=true&kind=research_plan&status=approved&limit=100")
      .then((res) => active && setPlans(res.data ?? []))
      .catch(() => active && setFailed(true));
    return () => {
      active = false;
    };
  }, []);

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("fields.title_change_source")}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-2">
        <NativeSelect
          aria-label={t("fields.title_change_source")}
          value={value}
          onChange={(e) => onSelect(plans.find((p) => p.id === e.target.value) ?? null)}
        >
          <option value="">{t("ui.unregisteredResearch")}</option>
          {plans.map((plan) => (
            <option key={plan.id} value={plan.id} {...bidiProps(requestTitle(plan))}>
              {plan.request_no} — {requestTitle(plan)}
            </option>
          ))}
        </NativeSelect>
        <p className="text-xs text-muted-foreground">
          {failed ? t("errors.generic") : plans.length === 0 ? t("ui.noApprovedPlans") : t("ui.titleChangeSourceHint")}
        </p>
      </CardContent>
    </Card>
  );
}
