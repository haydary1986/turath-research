"use client";

import { useParams } from "next/navigation";
import { useTranslations } from "next-intl";
import { Printer } from "lucide-react";
import { Button } from "@/components/ui/button";
import { RequireAuth } from "@/components/portal/require-auth";
import { RequestSummary } from "@/components/portal/request-summary";
import { useRequestDetail } from "@/components/portal/use-request-detail";

const COMMITTEE_SLOTS = [1, 2, 3];

function PrintForm() {
  const params = useParams<{ id: string }>();
  const t = useTranslations("portal");
  const { detail, loading, notFound } = useRequestDetail(params.id);

  if (loading) return <p className="text-center text-muted-foreground">{t("ui.loading")}</p>;
  if (notFound || !detail) {
    return <p className="text-center text-muted-foreground">{t("ui.notFound")}</p>;
  }
  const { request } = detail;

  return (
    <div className="space-y-4">
      <div className="flex justify-end gap-2 print:hidden">
        <Button onClick={() => window.print()}>
          <Printer className="size-4" />
          <span>{t("ui.print")}</span>
        </Button>
      </div>

      <div className="mx-auto bg-white p-10 text-black shadow print:p-0 print:shadow-none" style={{ width: "210mm", minHeight: "297mm" }}>
        <div className="mb-8 text-center">
          <h1 className="text-2xl font-bold">{t(`kinds.${request.kind}`)}</h1>
          <p className="text-sm">{t("title")}</p>
        </div>

        <RequestSummary request={request} />

        <div className="mt-12 space-y-8">
          {COMMITTEE_SLOTS.map((n) => (
            <div key={n} className="flex items-end gap-4 text-sm">
              <span className="w-8">{n}.</span>
              <span className="flex-1 border-b border-black" />
              <span className="w-32 border-b border-black text-center" />
            </div>
          ))}
        </div>

        <div className="mt-16 grid grid-cols-2 gap-8 text-center text-sm">
          <div className="h-28 rounded border border-dashed border-black pt-2">{t("fields.signer_name")}</div>
          <div className="h-28 rounded border border-dashed border-black pt-2">{t("fields.signer_title")}</div>
        </div>
      </div>
    </div>
  );
}

export default function PrintFormPage() {
  return (
    <RequireAuth>
      <PrintForm />
    </RequireAuth>
  );
}
