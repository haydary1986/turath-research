"use client";

import { useParams } from "next/navigation";
import { useTranslations } from "next-intl";
import { RequestForm } from "@/components/portal/request-form";
import { RequireAuth } from "@/components/portal/require-auth";
import { useRequestDetail } from "@/components/portal/use-request-detail";

function EditRequest() {
  const params = useParams<{ id: string }>();
  const t = useTranslations("portal");
  const { detail, loading, notFound } = useRequestDetail(params.id);

  if (loading) return <p className="text-center text-muted-foreground">{t("ui.loading")}</p>;
  if (notFound || !detail || !detail.can_edit) {
    return <p className="text-center text-muted-foreground">{t("ui.notFound")}</p>;
  }

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <h1 className="text-2xl font-bold">{t(`kinds.${detail.request.kind}`)}</h1>
      <RequestForm kind={detail.request.kind} existing={detail.request} />
    </div>
  );
}

export default function EditRequestPage() {
  return (
    <RequireAuth>
      <EditRequest />
    </RequireAuth>
  );
}
