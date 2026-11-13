"use client";

import { useEffect, useState, type ReactNode } from "react";
import { useLocale, useTranslations } from "next-intl";
import { ArrowDown, ArrowUp, Plus, Trash2 } from "lucide-react";
import { api } from "@/lib/api";
import { formatDateTime } from "@/lib/portal/types";
import { PUBLIC_API, SITE_LIMITS, type SiteContent, type SiteLink, type SiteService } from "@/lib/site/types";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/portal/form-controls";
import { CharCount } from "./char-count";
import { useSiteErrors } from "./use-site-errors";

type Message = { type: "ok" | "error"; text: string } | null;

function move<T>(list: T[], from: number, to: number): T[] {
  if (to < 0 || to >= list.length) return list;
  const next = [...list];
  const [item] = next.splice(from, 1);
  next.splice(to, 0, item);
  return next;
}

function replaceAt<T>(list: T[], index: number, value: T): T[] {
  return list.map((item, i) => (i === index ? value : item));
}

function Field({ id, label, children }: { id: string; label: string; children: ReactNode }) {
  return (
    <div className="space-y-1.5">
      <Label htmlFor={id}>{label}</Label>
      {children}
    </div>
  );
}

interface RowControlsProps {
  index: number;
  count: number;
  onMove: (to: number) => void;
  onRemove: () => void;
}

function RowControls({ index, count, onMove, onRemove }: RowControlsProps) {
  const t = useTranslations("portal.site.content");
  return (
    <div className="flex gap-1">
      <Button type="button" variant="ghost" size="icon-sm" aria-label={t("moveUp")} disabled={index === 0} onClick={() => onMove(index - 1)}>
        <ArrowUp aria-hidden />
      </Button>
      <Button type="button" variant="ghost" size="icon-sm" aria-label={t("moveDown")} disabled={index === count - 1} onClick={() => onMove(index + 1)}>
        <ArrowDown aria-hidden />
      </Button>
      <Button type="button" variant="ghost" size="icon-sm" aria-label={t("remove")} onClick={onRemove}>
        <Trash2 className="text-destructive" aria-hidden />
      </Button>
    </div>
  );
}

/** Edits the single landing-page document (hero, about, services, contact, quick links). */
export function SiteContentForm() {
  const t = useTranslations("portal.site");
  const tc = useTranslations("portal.site.content");
  const locale = useLocale();
  const toMessage = useSiteErrors();
  const [draft, setDraft] = useState<SiteContent | null>(null);
  const [loadError, setLoadError] = useState(false);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<Message>(null);

  useEffect(() => {
    let active = true;
    api
      .get<SiteContent>(PUBLIC_API.site)
      .then((res) => active && res.data && setDraft(res.data))
      .catch(() => active && setLoadError(true));
    return () => {
      active = false;
    };
  }, []);

  if (loadError) return <Alert variant="danger">{toMessage(null)}</Alert>;
  if (!draft) return <Skeleton className="h-96 w-full" />;

  const update = (patch: Partial<SiteContent>) => setDraft({ ...draft, ...patch });
  const setService = (i: number, patch: Partial<SiteService>) =>
    update({ services: replaceAt(draft.services, i, { ...draft.services[i], ...patch }) });
  const setLink = (i: number, patch: Partial<SiteLink>) =>
    update({ links: replaceAt(draft.links, i, { ...draft.links[i], ...patch }) });

  async function save() {
    if (!draft) return;
    setBusy(true);
    setMessage(null);
    try {
      const res = await api.put<SiteContent>("/api/v1/site", draft);
      if (res.data) setDraft(res.data);
      setMessage({ type: "ok", text: tc("saved") });
    } catch (err) {
      setMessage({ type: "error", text: toMessage(err) });
    } finally {
      setBusy(false);
    }
  }

  return (
    <form
      className="space-y-6"
      onSubmit={(e) => {
        e.preventDefault();
        void save();
      }}
    >
      <Card>
        <CardHeader>
          <CardTitle>{tc("hero")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <Field id="hero_title" label={tc("heroTitle")}>
            <Input id="hero_title" required maxLength={SITE_LIMITS.heroTitle} value={draft.hero_title} onChange={(e) => update({ hero_title: e.target.value })} />
          </Field>
          <Field id="hero_subtitle" label={tc("heroSubtitle")}>
            <Textarea id="hero_subtitle" rows={2} value={draft.hero_subtitle} onChange={(e) => update({ hero_subtitle: e.target.value })} />
            <CharCount value={draft.hero_subtitle} max={SITE_LIMITS.heroSubtitle} />
          </Field>
          <Field id="about" label={tc("about")}>
            <Textarea id="about" rows={8} aria-describedby="about_hint" value={draft.about} onChange={(e) => update({ about: e.target.value })} />
            <p id="about_hint" className="text-xs text-muted-foreground">{t("formatHint")}</p>
            <CharCount value={draft.about} max={SITE_LIMITS.about} />
          </Field>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex-row items-center justify-between">
          <CardTitle>{tc("services")}</CardTitle>
          <Button type="button" variant="outline" size="sm" disabled={draft.services.length >= SITE_LIMITS.services}
            onClick={() => update({ services: [...draft.services, { title: "", description: "" }] })}>
            <Plus aria-hidden />
            {tc("addService")}
          </Button>
        </CardHeader>
        <CardContent className="space-y-3">
          {draft.services.map((service, i) => (
            <fieldset key={i} className="space-y-3 rounded-lg border p-3">
              <legend className="sr-only">{tc("itemN", { n: i + 1 })}</legend>
              <div className="flex items-start gap-2">
                <div className="flex-1">
                  <Field id={`service_title_${i}`} label={tc("serviceTitle")}>
                    <Input id={`service_title_${i}`} required maxLength={SITE_LIMITS.serviceTitle} value={service.title} onChange={(e) => setService(i, { title: e.target.value })} />
                  </Field>
                </div>
                <RowControls index={i} count={draft.services.length}
                  onMove={(to) => update({ services: move(draft.services, i, to) })}
                  onRemove={() => update({ services: draft.services.filter((_, j) => j !== i) })} />
              </div>
              <Field id={`service_desc_${i}`} label={tc("serviceDescription")}>
                <Textarea id={`service_desc_${i}`} rows={2} value={service.description} onChange={(e) => setService(i, { description: e.target.value })} />
                <CharCount value={service.description} max={SITE_LIMITS.serviceDescription} />
              </Field>
            </fieldset>
          ))}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{tc("contact")}</CardTitle>
          <p className="text-sm text-muted-foreground">{tc("contactHint")}</p>
        </CardHeader>
        <CardContent className="grid gap-4 sm:grid-cols-2">
          <Field id="contact_email" label={tc("email")}>
            <Input id="contact_email" type="email" dir="ltr" maxLength={SITE_LIMITS.email} value={draft.contact.email}
              onChange={(e) => update({ contact: { ...draft.contact, email: e.target.value } })} />
          </Field>
          <Field id="contact_phone" label={tc("phone")}>
            <Input id="contact_phone" type="tel" dir="ltr" maxLength={SITE_LIMITS.phone} value={draft.contact.phone}
              onChange={(e) => update({ contact: { ...draft.contact, phone: e.target.value } })} />
          </Field>
          <Field id="contact_location" label={tc("location")}>
            <Input id="contact_location" maxLength={SITE_LIMITS.location} value={draft.contact.location}
              onChange={(e) => update({ contact: { ...draft.contact, location: e.target.value } })} />
          </Field>
          <Field id="contact_hours" label={tc("hours")}>
            <Input id="contact_hours" maxLength={SITE_LIMITS.hours} value={draft.contact.hours}
              onChange={(e) => update({ contact: { ...draft.contact, hours: e.target.value } })} />
          </Field>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex-row items-center justify-between">
          <CardTitle>{tc("links")}</CardTitle>
          <Button type="button" variant="outline" size="sm" disabled={draft.links.length >= SITE_LIMITS.links}
            onClick={() => update({ links: [...draft.links, { label: "", url: "https://" }] })}>
            <Plus aria-hidden />
            {tc("addLink")}
          </Button>
        </CardHeader>
        <CardContent className="space-y-3">
          {draft.links.map((link, i) => (
            <fieldset key={i} className="grid items-end gap-2 rounded-lg border p-3 sm:grid-cols-[1fr_1.4fr_auto]">
              <legend className="sr-only">{tc("itemN", { n: i + 1 })}</legend>
              <Field id={`link_label_${i}`} label={tc("linkLabel")}>
                <Input id={`link_label_${i}`} required maxLength={SITE_LIMITS.linkLabel} value={link.label} onChange={(e) => setLink(i, { label: e.target.value })} />
              </Field>
              <Field id={`link_url_${i}`} label={tc("linkUrl")}>
                <Input id={`link_url_${i}`} type="url" dir="ltr" required pattern="https://.+" maxLength={SITE_LIMITS.linkUrl} value={link.url} onChange={(e) => setLink(i, { url: e.target.value })} />
              </Field>
              <RowControls index={i} count={draft.links.length}
                onMove={(to) => update({ links: move(draft.links, i, to) })}
                onRemove={() => update({ links: draft.links.filter((_, j) => j !== i) })} />
            </fieldset>
          ))}
        </CardContent>
      </Card>

      <div className="sticky bottom-0 z-10 -mx-1 flex flex-wrap items-center gap-3 border-t bg-background/95 px-1 py-3 backdrop-blur">
        <Button type="submit" disabled={busy}>{busy ? tc("saving") : tc("save")}</Button>
        {draft.updated_at && <span className="text-xs text-muted-foreground">{tc("lastUpdated", { date: formatDateTime(draft.updated_at, locale) })}</span>}
        {message && <Alert variant={message.type === "ok" ? "success" : "danger"} className="w-full">{message.text}</Alert>}
      </div>
    </form>
  );
}
