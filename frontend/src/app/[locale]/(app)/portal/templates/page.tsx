"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslations } from "next-intl";
import { api } from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/portal/form-controls";
import { LetterView } from "@/components/portal/letter-view";
import { RequireAuth } from "@/components/portal/require-auth";
import { usePortalErrors } from "@/components/portal/use-portal-errors";
import { Alert } from "@/components/ui/alert";
import { FormTemplatesCard } from "@/components/portal/form-templates-card";
import { LETTER_PLACEHOLDERS, SAMPLE_LETTER_VARS, renderBody } from "@/lib/portal/letter";
import type { Attachment, LetterTemplate } from "@/lib/portal/types";

type Draft = Omit<LetterTemplate, "id" | "is_active"> & { id: string | null };

const EMPTY_DRAFT: Draft = {
  id: null,
  name: "",
  header_line_1: "",
  header_line_2: "",
  title_text: "كتاب شكر وتقدير",
  body_text: "",
  closing_text: "مع التقدير",
  signer_name: "",
  signer_title: "",
  background_file_id: null,
  signature_file_id: null,
};

const TEXT_FIELDS = ["header_line_1", "header_line_2", "title_text", "closing_text", "signer_name", "signer_title"] as const;

function toDraft(template: LetterTemplate): Draft {
  return {
    id: template.id,
    name: template.name,
    header_line_1: template.header_line_1,
    header_line_2: template.header_line_2,
    title_text: template.title_text,
    body_text: template.body_text,
    closing_text: template.closing_text,
    signer_name: template.signer_name,
    signer_title: template.signer_title,
    background_file_id: template.background_file_id,
    signature_file_id: template.signature_file_id,
  };
}

function Templates() {
  const t = useTranslations("portal");
  const toMessage = usePortalErrors();
  const [templates, setTemplates] = useState<LetterTemplate[]>([]);
  const [draft, setDraft] = useState<Draft>(EMPTY_DRAFT);
  const [message, setMessage] = useState<{ type: "ok" | "error"; text: string } | null>(null);
  const [busy, setBusy] = useState(false);
  const bodyRef = useRef<HTMLTextAreaElement>(null);

  const load = useCallback(async () => {
    const res = await api.get<LetterTemplate[]>("/api/v1/portal/templates");
    setTemplates(res.data ?? []);
  }, []);

  useEffect(() => {
    load().catch(() => undefined);
  }, [load]);

  function update<K extends keyof Draft>(key: K, value: Draft[K]) {
    setDraft((prev) => ({ ...prev, [key]: value }));
  }

  function insertPlaceholder(name: string) {
    const token = `{{${name}}}`;
    const el = bodyRef.current;
    const start = el?.selectionStart ?? draft.body_text.length;
    const end = el?.selectionEnd ?? draft.body_text.length;
    update("body_text", draft.body_text.slice(0, start) + token + draft.body_text.slice(end));
  }

  async function uploadAsset(role: "background" | "signature", file: File | undefined) {
    if (!file) return;
    setMessage(null);
    try {
      const res = await api.upload<Attachment>(`/api/v1/portal/template-assets?role=${role}`, file);
      if (res.data) update(role === "background" ? "background_file_id" : "signature_file_id", res.data.id);
    } catch (err) {
      setMessage({ type: "error", text: toMessage(err) });
    }
  }

  async function save() {
    setMessage(null);
    setBusy(true);
    try {
      const { id, ...payload } = draft;
      const res = id
        ? await api.put<LetterTemplate>(`/api/v1/portal/templates/${id}`, payload)
        : await api.post<LetterTemplate>("/api/v1/portal/templates", payload);
      if (res.data) setDraft(toDraft(res.data));
      await load();
      setMessage({ type: "ok", text: t("templates.saved") });
    } catch (err) {
      setMessage({ type: "error", text: toMessage(err) });
    } finally {
      setBusy(false);
    }
  }

  async function activate(id: string) {
    setMessage(null);
    try {
      await api.post(`/api/v1/portal/templates/${id}/activate`, {});
      await load();
      setMessage({ type: "ok", text: t("templates.activated") });
    } catch (err) {
      setMessage({ type: "error", text: toMessage(err) });
    }
  }

  const preview = {
    letter_no: "1245/2026",
    issued_date: new Date().toISOString().slice(0, 10),
    recipient_name: SAMPLE_LETTER_VARS.researcher_name,
    college_name: SAMPLE_LETTER_VARS.college,
    department_name: SAMPLE_LETTER_VARS.department,
    header_line_1: draft.header_line_1,
    header_line_2: draft.header_line_2,
    title_text: draft.title_text,
    body_text: renderBody(draft.body_text, SAMPLE_LETTER_VARS),
    closing_text: draft.closing_text,
    signer_name: draft.signer_name,
    signer_title: draft.signer_title,
    background_file_id: draft.background_file_id,
    signature_file_id: draft.signature_file_id,
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{t("templates.title")}</h1>
        <p className="text-sm text-muted-foreground">{t("templates.intro")}</p>
      </div>

      {message && (
        <Alert variant={message.type === "ok" ? "success" : "danger"}>{message.text}</Alert>
      )}

      <FormTemplatesCard />

      <Card>
        <CardContent className="flex flex-wrap items-center gap-2 pt-4">
          {templates.length === 0 && <span className="text-sm text-muted-foreground">{t("templates.empty")}</span>}
          {templates.map((tpl) => (
            <div key={tpl.id} className="flex items-center gap-2 rounded-lg border p-2">
              <button type="button" className="text-sm font-medium hover:underline" onClick={() => setDraft(toDraft(tpl))}>
                {tpl.name}
              </button>
              {tpl.is_active ? (
                <Badge>{t("templates.active")}</Badge>
              ) : (
                <Button size="xs" variant="outline" onClick={() => activate(tpl.id)}>
                  {t("templates.activate")}
                </Button>
              )}
            </div>
          ))}
          <Button size="sm" variant="secondary" onClick={() => setDraft(EMPTY_DRAFT)}>
            {t("templates.new")}
          </Button>
        </CardContent>
      </Card>

      <div className="grid gap-6 xl:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>{draft.id ? t("templates.edit") : t("templates.new")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="tpl-name">{t("templates.name")} *</Label>
              <Input id="tpl-name" value={draft.name} onChange={(e) => update("name", e.target.value)} />
            </div>

            <div className="grid gap-4 md:grid-cols-2">
              {TEXT_FIELDS.map((field) => (
                <div key={field} className="space-y-2">
                  <Label htmlFor={`tpl-${field}`}>{t(`fields.${field}`)}</Label>
                  <Input id={`tpl-${field}`} value={draft[field]} onChange={(e) => update(field, e.target.value)} />
                </div>
              ))}
            </div>

            <div className="space-y-2">
              <Label htmlFor="tpl-body">{t("fields.body_text")} *</Label>
              <Textarea
                id="tpl-body"
                ref={bodyRef}
                className="min-h-40"
                value={draft.body_text}
                onChange={(e) => update("body_text", e.target.value)}
              />
              <p className="text-xs text-muted-foreground">{t("templates.placeholders")}</p>
              <div className="flex flex-wrap gap-1">
                {LETTER_PLACEHOLDERS.map((name) => (
                  <Button key={name} type="button" size="xs" variant="outline" onClick={() => insertPlaceholder(name)}>
                    {t(`templates.ph.${name}`)}
                  </Button>
                ))}
              </div>
            </div>

            <div className="grid gap-4 md:grid-cols-2">
              <AssetPicker
                label={t("templates.background")}
                uploadLabel={t("templates.uploadImage")}
                attached={!!draft.background_file_id}
                onFile={(file) => uploadAsset("background", file)}
                onClear={() => update("background_file_id", null)}
              />
              <AssetPicker
                label={t("templates.signature")}
                uploadLabel={t("templates.uploadImage")}
                attached={!!draft.signature_file_id}
                onFile={(file) => uploadAsset("signature", file)}
                onClear={() => update("signature_file_id", null)}
              />
            </div>

            <Button disabled={busy} onClick={save}>
              {t("templates.save")}
            </Button>
          </CardContent>
        </Card>

        <div className="space-y-2">
          <h2 className="text-sm font-medium text-muted-foreground">
            {t("templates.preview")} ({t("templates.sample")})
          </h2>
          <div className="overflow-auto rounded-lg border bg-muted/30 p-2">
            <div style={{ zoom: 0.5 }}>
              <LetterView letter={preview} />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

interface AssetPickerProps {
  label: string;
  uploadLabel: string;
  attached: boolean;
  onFile: (file: File | undefined) => void;
  onClear: () => void;
}

function AssetPicker({ label, uploadLabel, attached, onFile, onClear }: AssetPickerProps) {
  const input = useRef<HTMLInputElement>(null);
  return (
    <div className="space-y-2">
      <Label>{label}</Label>
      <input
        ref={input}
        type="file"
        accept="image/png,image/jpeg"
        className="hidden"
        onChange={(e) => {
          onFile(e.target.files?.[0]);
          e.target.value = "";
        }}
      />
      <div className="flex items-center gap-2">
        <Button type="button" variant="outline" size="sm" onClick={() => input.current?.click()}>
          {uploadLabel}
        </Button>
        {attached && (
          <Button type="button" variant="ghost" size="sm" onClick={onClear}>
            ✕
          </Button>
        )}
      </div>
    </div>
  );
}

export default function TemplatesPage() {
  return (
    <RequireAuth>
      <Templates />
    </RequireAuth>
  );
}
