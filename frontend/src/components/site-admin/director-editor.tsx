"use client";

import { useRef, useState } from "react";
import { useTranslations } from "next-intl";
import { ImageUp, Trash2, UserRound } from "lucide-react";
import { api } from "@/lib/api";
import { PUBLIC_API, SITE_ADMIN_API, SITE_LIMITS, type SiteContent, type SiteDirector } from "@/lib/site/types";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/portal/form-controls";
import { CharCount } from "./char-count";
import { Field } from "./editor-parts";
import { useSiteErrors } from "./use-site-errors";

interface DirectorEditorProps {
  director: SiteDirector;
  onChange: (director: SiteDirector) => void;
}

type Notice = { type: "ok" | "error"; text: string } | null;

/** Director's message: text fields saved with the page, photo uploaded and removed immediately. */
export function DirectorEditor({ director, onChange }: DirectorEditorProps) {
  const t = useTranslations("siteEditor.director");
  const tp = useTranslations("portal.site");
  const toMessage = useSiteErrors();
  const fileInput = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState<Notice>(null);
  const set = (patch: Partial<SiteDirector>) => onChange({ ...director, ...patch });

  async function photoAction(run: () => Promise<SiteContent | undefined>, okText: string) {
    setBusy(true);
    setNotice(null);
    try {
      const saved = await run();
      // Only the photo changes server-side; unsaved text edits stay in the draft.
      onChange({ ...director, photo_id: saved?.director.photo_id ?? "" });
      setNotice({ type: "ok", text: okText });
    } catch (err) {
      setNotice({ type: "error", text: toMessage(err) });
    } finally {
      setBusy(false);
    }
  }

  const upload = (file: File) =>
    photoAction(async () => (await api.upload<SiteContent>(SITE_ADMIN_API.directorPhoto, file)).data, t("uploaded"));
  const remove = () => {
    if (!window.confirm(t("removeConfirm"))) return;
    void photoAction(async () => (await api.delete<SiteContent>(SITE_ADMIN_API.directorPhoto)).data, t("removed"));
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("title")}</CardTitle>
        <p className="text-sm text-muted-foreground">{t("hint")}</p>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-4 sm:grid-cols-2">
          <Field id="director_title" label={t("blockTitle")}>
            <Input id="director_title" maxLength={SITE_LIMITS.directorTitle} value={director.title} onChange={(e) => set({ title: e.target.value })} />
          </Field>
          <Field id="director_name" label={t("name")}>
            <Input id="director_name" maxLength={SITE_LIMITS.directorName} value={director.name} onChange={(e) => set({ name: e.target.value })} />
          </Field>
          <Field id="director_position" label={t("position")}>
            <Input id="director_position" maxLength={SITE_LIMITS.directorPosition} value={director.position}
              onChange={(e) => set({ position: e.target.value })} />
          </Field>
        </div>
        <Field id="director_message" label={t("message")} hint={tp("formatHint")}>
          <Textarea id="director_message" rows={6} value={director.message} onChange={(e) => set({ message: e.target.value })} />
          <CharCount value={director.message} max={SITE_LIMITS.directorMessage} />
        </Field>

        <div className="flex flex-wrap items-center gap-4 rounded-lg border p-3">
          {director.photo_id ? (
            // eslint-disable-next-line @next/next/no-img-element -- small preview of an uploaded file
            <img src={PUBLIC_API.directorPhoto(director.photo_id)} alt="" className="size-20 rounded-xl object-cover" />
          ) : (
            <span className="flex size-20 items-center justify-center rounded-xl bg-muted text-muted-foreground" aria-hidden>
              <UserRound className="size-8" />
            </span>
          )}
          <div className="min-w-0 flex-1 space-y-2">
            <p className="text-sm font-semibold">{t("photo")}</p>
            <p className="text-xs text-muted-foreground">{t("photoHint")}</p>
            <div className="flex flex-wrap gap-2">
              <input
                ref={fileInput}
                type="file"
                accept="image/png,image/jpeg"
                className="hidden"
                aria-label={t("upload")}
                onChange={(e) => {
                  const file = e.target.files?.[0];
                  e.target.value = "";
                  if (file) void upload(file);
                }}
              />
              <Button type="button" variant="outline" size="sm" disabled={busy} onClick={() => fileInput.current?.click()}>
                <ImageUp aria-hidden />
                {director.photo_id ? t("replace") : t("upload")}
              </Button>
              {director.photo_id && (
                <Button type="button" variant="ghost" size="sm" disabled={busy} onClick={remove}>
                  <Trash2 className="text-destructive" aria-hidden />
                  {t("remove")}
                </Button>
              )}
            </div>
          </div>
        </div>
        {notice && <Alert variant={notice.type === "ok" ? "success" : "danger"}>{notice.text}</Alert>}
      </CardContent>
    </Card>
  );
}
