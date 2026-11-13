"use client";

import { useParams } from "next/navigation";
import { useTranslations } from "next-intl";
import { Alert } from "@/components/ui/alert";
import { OfficialFormButton } from "@/components/portal/official-form";
import { RequestForm } from "@/components/portal/request-form";
import { RequireAuth } from "@/components/portal/require-auth";
import { REQUEST_KINDS, type RequestKind } from "@/lib/portal/types";

export default function NewRequestPage() {
  const params = useParams<{ kind: string }>();
  const t = useTranslations("portal");
  const kind = REQUEST_KINDS.find((k) => k === params.kind) as RequestKind | undefined;

  if (!kind) return <p className="text-center text-muted-foreground">{t("ui.notFound")}</p>;

  return (
    <RequireAuth>
      <div className="mx-auto max-w-3xl space-y-6">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h1 className="text-2xl font-bold">{t(`kinds.${kind}`)}</h1>
          <OfficialFormButton kind={kind} />
        </div>
        {kind === "research_plan" && <Alert variant="info">{t("forms.planStepsIntro")}</Alert>}
        <RequestForm kind={kind} />
      </div>
    </RequireAuth>
  );
}
