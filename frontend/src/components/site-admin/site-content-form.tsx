"use client";

import { useEffect, useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import { api } from "@/lib/api";
import { formatDateTime } from "@/lib/portal/types";
import { PUBLIC_API, SITE_ADMIN_API, SITE_LIMITS, type SiteContent } from "@/lib/site/types";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/portal/form-controls";
import { CharCount } from "./char-count";
import { DirectorEditor } from "./director-editor";
import { Field } from "./editor-parts";
import { FaqEditor } from "./faq-editor";
import { LinksEditor } from "./links-editor";
import { ServicesEditor } from "./services-editor";
import { SocialsEditor } from "./socials-editor";
import { StatsEditor } from "./stats-editor";
import { useSiteErrors } from "./use-site-errors";

type Message = { type: "ok" | "error"; text: string } | null;

/** Edits the single landing-page document: hero, services, page blocks and footer. */
export function SiteContentForm() {
  const t = useTranslations("portal.site");
  const tc = useTranslations("portal.site.content");
  const te = useTranslations("siteEditor");
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

  async function save() {
    if (!draft) return;
    setBusy(true);
    setMessage(null);
    try {
      const res = await api.put<SiteContent>(SITE_ADMIN_API.content, draft);
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
      <Tabs defaultValue="content">
        <TabsList>
          <TabsTrigger value="content">{te("tabs.content")}</TabsTrigger>
          <TabsTrigger value="blocks">{te("tabs.blocks")}</TabsTrigger>
          <TabsTrigger value="footer">{te("tabs.footer")}</TabsTrigger>
        </TabsList>

        <TabsContent value="content" className="space-y-6 pt-4">
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
              <Field id="about" label={tc("about")} hint={t("formatHint")}>
                <Textarea id="about" rows={8} value={draft.about} onChange={(e) => update({ about: e.target.value })} />
                <CharCount value={draft.about} max={SITE_LIMITS.about} />
              </Field>
            </CardContent>
          </Card>
          <ServicesEditor services={draft.services} onChange={(services) => update({ services })} />
        </TabsContent>

        <TabsContent value="blocks" className="space-y-6 pt-4">
          <StatsEditor stats={draft.stats} onChange={(stats) => update({ stats })} />
          <Card>
            <CardHeader>
              <CardTitle>{te("publications.title")}</CardTitle>
              <p className="text-sm text-muted-foreground">{te("publications.hint")}</p>
            </CardHeader>
            <CardContent>
              <label className="flex items-center gap-2 text-sm">
                <input type="checkbox" className="size-4 accent-primary" checked={!draft.hide_publications}
                  onChange={(e) => update({ hide_publications: !e.target.checked })} />
                {te("publications.show")}
              </label>
            </CardContent>
          </Card>
          <DirectorEditor director={draft.director} onChange={(director) => update({ director })} />
          <FaqEditor items={draft.faq} onChange={(faq) => update({ faq })} />
        </TabsContent>

        <TabsContent value="footer" className="space-y-6 pt-4">
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
          <LinksEditor idPrefix="college" title={te("colleges.title")} hint={te("colleges.hint")} addLabel={te("colleges.add")}
            labelLabel={te("colleges.name")} urlLabel={te("colleges.url")} max={SITE_LIMITS.colleges}
            links={draft.colleges} onChange={(colleges) => update({ colleges })} />
          <SocialsEditor socials={draft.socials} onChange={(socials) => update({ socials })} />
          <LinksEditor idPrefix="link" title={tc("links")} addLabel={tc("addLink")} labelLabel={tc("linkLabel")}
            urlLabel={tc("linkUrl")} max={SITE_LIMITS.links} links={draft.links} onChange={(links) => update({ links })} />
        </TabsContent>
      </Tabs>

      <div className="sticky bottom-0 z-10 -mx-1 flex flex-wrap items-center gap-3 border-t bg-background/95 px-1 py-3 backdrop-blur">
        <Button type="submit" disabled={busy}>{busy ? tc("saving") : tc("save")}</Button>
        <span className="text-xs text-muted-foreground">{te("unsavedHint")}</span>
        {draft.updated_at && <span className="text-xs text-muted-foreground">{tc("lastUpdated", { date: formatDateTime(draft.updated_at, locale) })}</span>}
        {message && <Alert variant={message.type === "ok" ? "success" : "danger"} className="w-full whitespace-pre-line">{message.text}</Alert>}
      </div>
    </form>
  );
}
