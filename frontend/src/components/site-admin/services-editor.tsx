"use client";

import { useTranslations } from "next-intl";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { NativeSelect, Textarea } from "@/components/portal/form-controls";
import { ServiceIcon } from "@/components/site/service-icon";
import { SERVICE_ICONS, SITE_LIMITS, isSafeServiceLink, type SiteService } from "@/lib/site/types";
import { CharCount } from "./char-count";
import { Field, ListHeader, RowControls, move, removeAt, replaceAt } from "./editor-parts";

interface ServicesEditorProps {
  services: SiteService[];
  onChange: (services: SiteService[]) => void;
}

/** Service cards: title, description, icon and an internal link. */
export function ServicesEditor({ services, onChange }: ServicesEditorProps) {
  const tc = useTranslations("portal.site.content");
  const te = useTranslations("siteEditor");
  const set = (i: number, patch: Partial<SiteService>) => onChange(replaceAt(services, i, { ...services[i], ...patch }));

  return (
    <Card>
      <ListHeader
        title={tc("services")}
        addLabel={tc("addService")}
        canAdd={services.length < SITE_LIMITS.services}
        onAdd={() => onChange([...services, { title: "", description: "", icon: "", link: "" }])}
      />
      <CardContent className="space-y-3">
        {services.map((service, i) => {
          const linkInvalid = service.link !== "" && !isSafeServiceLink(service.link);
          return (
            <fieldset key={i} className="space-y-3 rounded-lg border p-3">
              <legend className="sr-only">{tc("itemN", { n: i + 1 })}</legend>
              <div className="flex items-start gap-2">
                <span className="mt-6 flex size-9 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground" aria-hidden>
                  <ServiceIcon name={service.icon} className="size-5" />
                </span>
                <div className="flex-1">
                  <Field id={`service_title_${i}`} label={tc("serviceTitle")}>
                    <Input id={`service_title_${i}`} required maxLength={SITE_LIMITS.serviceTitle} value={service.title}
                      onChange={(e) => set(i, { title: e.target.value })} />
                  </Field>
                </div>
                <RowControls index={i} count={services.length}
                  onMove={(to) => onChange(move(services, i, to))} onRemove={() => onChange(removeAt(services, i))} />
              </div>
              <Field id={`service_desc_${i}`} label={tc("serviceDescription")}>
                <Textarea id={`service_desc_${i}`} rows={2} value={service.description}
                  onChange={(e) => set(i, { description: e.target.value })} />
                <CharCount value={service.description} max={SITE_LIMITS.serviceDescription} />
              </Field>
              <div className="grid gap-3 sm:grid-cols-[12rem_1fr]">
                <Field id={`service_icon_${i}`} label={te("serviceIcon")}>
                  <NativeSelect id={`service_icon_${i}`} value={service.icon} onChange={(e) => set(i, { icon: e.target.value })}>
                    <option value="">{te("icons.default")}</option>
                    {SERVICE_ICONS.map((key) => <option key={key} value={key}>{te(`icons.${key}`)}</option>)}
                  </NativeSelect>
                </Field>
                <Field id={`service_link_${i}`} label={te("serviceLink")} hint={te("serviceLinkHint")}>
                  <Input id={`service_link_${i}`} dir="ltr" maxLength={SITE_LIMITS.serviceLink} value={service.link}
                    placeholder="/portal/new/research_plan" aria-invalid={linkInvalid}
                    onChange={(e) => set(i, { link: e.target.value.trim() })} />
                </Field>
              </div>
            </fieldset>
          );
        })}
      </CardContent>
    </Card>
  );
}
