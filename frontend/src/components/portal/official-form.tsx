"use client";

import { useCallback, useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { FileDown } from "lucide-react";
import { api, downloadFile } from "@/lib/api";
import { Button } from "@/components/ui/button";
import type { RequestKind } from "@/lib/portal/types";

export interface FormTemplate {
  kind: RequestKind;
  file_id: string;
  original_name: string;
  updated_at: string;
}

/** Loads the official blank forms uploaded by scientific affairs. */
export function useFormTemplates(): { forms: FormTemplate[]; reload: () => void } {
  const [forms, setForms] = useState<FormTemplate[]>([]);
  const [version, setVersion] = useState(0);

  useEffect(() => {
    let active = true;
    api
      .get<FormTemplate[]>("/api/v1/portal/form-templates")
      .then((res) => active && setForms(res.data ?? []))
      .catch(() => undefined);
    return () => {
      active = false;
    };
  }, [version]);

  const reload = useCallback(() => setVersion((v) => v + 1), []);
  return { forms, reload };
}

/** Download button for the official blank form of one kind; renders nothing when none is uploaded. */
export function OfficialFormButton({ kind }: { kind: RequestKind }) {
  const t = useTranslations("portal.forms");
  const { forms } = useFormTemplates();
  const [busy, setBusy] = useState(false);
  const form = forms.find((f) => f.kind === kind);
  if (!form) return null;

  async function download() {
    if (!form) return;
    setBusy(true);
    try {
      await downloadFile(`/api/v1/portal/files/${form.file_id}`, form.original_name);
    } finally {
      setBusy(false);
    }
  }

  return (
    <Button type="button" variant="outline" size="sm" disabled={busy} onClick={download}>
      <FileDown className="size-4" aria-hidden />
      {t("downloadBlank")}
    </Button>
  );
}
