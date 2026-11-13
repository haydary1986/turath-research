"use client";

import { useRef, useState } from "react";
import { useTranslations } from "next-intl";
import { FileUp, Trash2 } from "lucide-react";
import { api, downloadFile } from "@/lib/api";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { REQUEST_KINDS, type RequestKind } from "@/lib/portal/types";
import { useFormTemplates } from "./official-form";
import { usePortalErrors } from "./use-portal-errors";

/** Staff manage the official blank PDF form applicants download for each request kind. */
export function FormTemplatesCard() {
  const t = useTranslations("portal");
  const tf = useTranslations("portal.forms");
  const toMessage = usePortalErrors();
  const { forms, reload } = useFormTemplates();
  const [busyKind, setBusyKind] = useState<RequestKind | null>(null);
  const [error, setError] = useState("");
  const inputs = useRef<Partial<Record<RequestKind, HTMLInputElement | null>>>({});

  async function upload(kind: RequestKind, file: File) {
    setError("");
    setBusyKind(kind);
    try {
      await api.upload(`/api/v1/portal/form-templates/${kind}`, file);
      reload();
    } catch (err) {
      setError(toMessage(err));
    } finally {
      setBusyKind(null);
      const input = inputs.current[kind];
      if (input) input.value = "";
    }
  }

  async function remove(kind: RequestKind) {
    if (!window.confirm(tf("deleteConfirm"))) return;
    setError("");
    setBusyKind(kind);
    try {
      await api.delete(`/api/v1/portal/form-templates/${kind}`);
      reload();
    } catch (err) {
      setError(toMessage(err));
    } finally {
      setBusyKind(null);
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{tf("title")}</CardTitle>
        <p className="text-sm text-muted-foreground">{tf("intro")}</p>
      </CardHeader>
      <CardContent className="space-y-3">
        {error && <Alert variant="danger">{error}</Alert>}
        {REQUEST_KINDS.map((kind) => {
          const form = forms.find((f) => f.kind === kind);
          return (
            <div key={kind} className="flex flex-wrap items-center justify-between gap-3 rounded-lg border p-3">
              <div className="min-w-0">
                <p className="font-medium">{t(`kinds.${kind}`)}</p>
                {form ? (
                  <button
                    type="button"
                    className="truncate text-sm text-primary hover:underline"
                    onClick={() => downloadFile(`/api/v1/portal/files/${form.file_id}`, form.original_name)}
                  >
                    {form.original_name}
                  </button>
                ) : (
                  <p className="text-sm text-muted-foreground">{tf("none")}</p>
                )}
              </div>
              <div className="flex gap-2">
                <input
                  ref={(el) => {
                    inputs.current[kind] = el;
                  }}
                  type="file"
                  accept="application/pdf"
                  className="hidden"
                  onChange={(e) => e.target.files?.[0] && upload(kind, e.target.files[0])}
                />
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={busyKind === kind}
                  onClick={() => inputs.current[kind]?.click()}
                >
                  <FileUp className="size-4" aria-hidden />
                  {form ? tf("replace") : tf("upload")}
                </Button>
                {form && (
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    aria-label={t("ui.remove")}
                    disabled={busyKind === kind}
                    onClick={() => remove(kind)}
                  >
                    <Trash2 className="size-3.5 text-destructive" />
                  </Button>
                )}
              </div>
            </div>
          );
        })}
      </CardContent>
    </Card>
  );
}
