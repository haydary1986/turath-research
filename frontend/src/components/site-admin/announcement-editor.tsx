"use client";

import { useRef, useState } from "react";
import { useTranslations } from "next-intl";
import { FileUp, Paperclip, Trash2 } from "lucide-react";
import { api, downloadFile } from "@/lib/api";
import {
  ANNOUNCEMENT_LANGS,
  ANNOUNCEMENT_TYPES,
  SITE_LIMITS,
  type AnnouncementInput,
  type StaffAnnouncement,
} from "@/lib/site/types";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { NativeSelect, Textarea } from "@/components/portal/form-controls";
import { Bidi } from "@/components/i18n/bidi";
import { dirOf } from "@/lib/i18n/text-lang";
import { CharCount } from "./char-count";
import { useSiteErrors } from "./use-site-errors";

const EMPTY: AnnouncementInput = { type: "circular", lang: "ar", title: "", body: "", event_date: "", reference_no: "" };

function toInput(a: StaffAnnouncement): AnnouncementInput {
  return {
    type: a.type,
    lang: a.lang ?? "ar",
    title: a.title,
    body: a.body,
    event_date: a.event_date ?? "",
    reference_no: a.reference_no ?? "",
  };
}

interface AnnouncementEditorProps {
  /** null creates a new announcement. */
  announcement: StaffAnnouncement | null;
  onSaved: (saved: StaffAnnouncement) => void;
  onClose: () => void;
}

/** Create/edit form for one announcement, with its single attachment once saved. */
export function AnnouncementEditor({ announcement, onSaved, onClose }: AnnouncementEditorProps) {
  const t = useTranslations("portal.site.ann");
  const ts = useTranslations("portal.site");
  const types = useTranslations("site.types");
  const toMessage = useSiteErrors();
  const [current, setCurrent] = useState<StaffAnnouncement | null>(announcement);
  const [form, setForm] = useState<AnnouncementInput>(announcement ? toInput(announcement) : EMPTY);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<{ type: "ok" | "error"; text: string } | null>(null);
  const fileInput = useRef<HTMLInputElement | null>(null);

  const set = (patch: Partial<AnnouncementInput>) => setForm({ ...form, ...patch });

  async function run(action: () => Promise<StaffAnnouncement | undefined>, okText?: string) {
    setBusy(true);
    setMessage(null);
    try {
      const saved = await action();
      if (saved) {
        setCurrent(saved);
        onSaved(saved);
      }
      if (okText) setMessage({ type: "ok", text: okText });
    } catch (err) {
      setMessage({ type: "error", text: toMessage(err) });
    } finally {
      setBusy(false);
    }
  }

  const save = () =>
    run(async () => {
      const res = current
        ? await api.put<StaffAnnouncement>(`/api/v1/announcements/${current.id}`, form)
        : await api.post<StaffAnnouncement>("/api/v1/announcements", form);
      return res.data;
    }, current?.published ? t("savedPublished") : t("savedDraft"));

  const upload = (file: File) =>
    run(async () => {
      if (!current) return undefined;
      try {
        return (await api.upload<StaffAnnouncement>(`/api/v1/announcements/${current.id}/attachment`, file)).data;
      } finally {
        if (fileInput.current) fileInput.current.value = "";
      }
    });

  const removeAttachment = () => {
    if (!current || !window.confirm(t("removeAttachmentConfirm"))) return;
    void run(async () => (await api.delete<StaffAnnouncement>(`/api/v1/announcements/${current.id}/attachment`)).data);
  };

  return (
    <Card className="border-t-4 border-t-primary">
      <CardHeader>
        <CardTitle>{current ? t("editTitle") : t("newTitle")}</CardTitle>
      </CardHeader>
      <CardContent>
        <form
          className="space-y-4"
          onSubmit={(e) => {
            e.preventDefault();
            void save();
          }}
        >
          <div className="grid gap-4 sm:grid-cols-[12rem_12rem_1fr]">
            <div className="space-y-1.5">
              <Label htmlFor="ann_type">{t("type")}</Label>
              <NativeSelect id="ann_type" className="h-9" value={form.type} onChange={(e) => set({ type: e.target.value as AnnouncementInput["type"] })}>
                {ANNOUNCEMENT_TYPES.map((type) => <option key={type} value={type}>{types(type)}</option>)}
              </NativeSelect>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="ann_lang">{t("lang")}</Label>
              <NativeSelect id="ann_lang" className="h-9" aria-describedby="ann_lang_hint" value={form.lang}
                onChange={(e) => set({ lang: e.target.value as AnnouncementInput["lang"] })}>
                {ANNOUNCEMENT_LANGS.map((lang) => <option key={lang} value={lang}>{t(`langs.${lang}`)}</option>)}
              </NativeSelect>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="ann_title">{t("title")}</Label>
              <Input id="ann_title" required lang={form.lang} dir={dirOf(form.lang)} maxLength={SITE_LIMITS.title} value={form.title}
                onChange={(e) => set({ title: e.target.value })} />
            </div>
            <p id="ann_lang_hint" className="text-xs text-muted-foreground sm:col-span-3">{t("langHint")}</p>
          </div>
          {form.type === "circular" ? (
            <div className="space-y-1.5 sm:max-w-xs">
              <Label htmlFor="ann_ref">{t("referenceNo")}</Label>
              <Input id="ann_ref" dir="auto" maxLength={SITE_LIMITS.referenceNo} value={form.reference_no} onChange={(e) => set({ reference_no: e.target.value })} />
            </div>
          ) : (
            <div className="space-y-1.5 sm:max-w-xs">
              <Label htmlFor="ann_date">{t("eventDate")}</Label>
              <Input id="ann_date" type="date" dir="ltr" value={form.event_date} onChange={(e) => set({ event_date: e.target.value })} />
            </div>
          )}
          <div className="space-y-1.5">
            <Label htmlFor="ann_body">{t("body")}</Label>
            <Textarea id="ann_body" rows={10} lang={form.lang} dir={dirOf(form.lang)} aria-describedby="ann_body_hint" value={form.body} onChange={(e) => set({ body: e.target.value })} />
            <p id="ann_body_hint" className="text-xs text-muted-foreground">{ts("formatHint")}</p>
            <CharCount value={form.body} max={SITE_LIMITS.body} />
          </div>

          <div className="space-y-2 rounded-lg border border-dashed p-3">
            <p className="text-sm font-medium">{t("attachment")}</p>
            {!current ? (
              <p className="text-xs text-muted-foreground">{t("saveFirst")}</p>
            ) : (
              <div className="flex flex-wrap items-center gap-2">
                {current.attachment && (
                  <button
                    type="button"
                    className="inline-flex max-w-full items-center gap-1 truncate text-sm text-primary hover:underline"
                    onClick={() => current.attachment && downloadFile(`/api/v1/portal/files/${current.attachment.id}`, current.attachment.name)}
                  >
                    <Paperclip className="size-4 shrink-0" aria-hidden />
                    <Bidi text={current.attachment.name} className="truncate" />
                  </button>
                )}
                <input
                  ref={fileInput}
                  type="file"
                  accept="application/pdf,image/png,image/jpeg"
                  className="hidden"
                  aria-label={t("uploadAttachment")}
                  onChange={(e) => e.target.files?.[0] && upload(e.target.files[0])}
                />
                <Button type="button" variant="outline" size="sm" disabled={busy} onClick={() => fileInput.current?.click()}>
                  <FileUp aria-hidden />
                  {current.attachment ? t("replaceAttachment") : t("uploadAttachment")}
                </Button>
                {current.attachment && (
                  <Button type="button" variant="ghost" size="sm" disabled={busy} onClick={removeAttachment}>
                    <Trash2 className="text-destructive" aria-hidden />
                    {t("removeAttachment")}
                  </Button>
                )}
                <p className="w-full text-xs text-muted-foreground">{t("attachmentHint")}</p>
              </div>
            )}
          </div>

          {message && <Alert variant={message.type === "ok" ? "success" : "danger"}>{message.text}</Alert>}
          <div className="flex gap-2">
            <Button type="submit" disabled={busy}>{busy ? t("saving") : t("save")}</Button>
            <Button type="button" variant="outline" onClick={onClose}>{t("cancel")}</Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
