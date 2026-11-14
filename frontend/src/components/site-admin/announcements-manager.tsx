"use client";

import { useCallback, useEffect, useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import { ExternalLink, Paperclip, Pin, Plus } from "lucide-react";
import { api } from "@/lib/api";
import { FAMILY_TINT } from "@/lib/portal/status";
import { formatDate } from "@/lib/portal/types";
import { ANNOUNCEMENT_TYPES, type StaffAnnouncement } from "@/lib/site/types";
import { cn } from "@/lib/utils";
import { Link } from "@/i18n/routing";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { NativeSelect } from "@/components/portal/form-controls";
import { AnnouncementTypeBadge } from "@/components/site/announcement-meta";
import { Bidi } from "@/components/i18n/bidi";
import { AnnouncementEditor } from "./announcement-editor";
import { useSiteErrors } from "./use-site-errors";

const PAGE_SIZE = 20;
type StatusFilter = "" | "published" | "draft";
/** null: editor closed; "new": creating; otherwise the announcement being edited. */
type Editing = null | "new" | StaffAnnouncement;

interface ListState {
  key: string;
  items: StaffAnnouncement[];
  total: number;
}

/** Staff list of announcements: filter, create/edit, publish, pin, delete. */
export function AnnouncementsManager({ initialEditId }: { initialEditId?: string }) {
  const t = useTranslations("portal.site.ann");
  const types = useTranslations("site.types");
  const tc = useTranslations("common");
  const locale = useLocale();
  const toMessage = useSiteErrors();
  const [type, setType] = useState("");
  const [status, setStatus] = useState<StatusFilter>("");
  const [page, setPage] = useState(1);
  const [version, setVersion] = useState(0);
  const [list, setList] = useState<ListState | null>(null);
  const [editing, setEditing] = useState<Editing>(null);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [error, setError] = useState("");

  const params = new URLSearchParams({ page: String(page), limit: String(PAGE_SIZE) });
  if (type) params.set("type", type);
  if (status) params.set("status", status);
  const key = `${params}#${version}`;
  const reload = useCallback(() => setVersion((v) => v + 1), []);

  useEffect(() => {
    let active = true;
    api
      .get<StaffAnnouncement[]>(`/api/v1/announcements?${key.split("#")[0]}`)
      .then((res) => active && setList({ key, items: res.data ?? [], total: res.meta?.total ?? 0 }))
      .catch((err: unknown) => active && setError(toMessage(err)));
    return () => {
      active = false;
    };
  }, [key, toMessage]);

  useEffect(() => {
    if (!initialEditId) return;
    let active = true;
    api
      .get<StaffAnnouncement>(`/api/v1/announcements/${encodeURIComponent(initialEditId)}`)
      .then((res) => active && res.data && setEditing(res.data))
      .catch((err: unknown) => active && setError(toMessage(err)));
    return () => {
      active = false;
    };
  }, [initialEditId, toMessage]);

  async function act(item: StaffAnnouncement, action: () => Promise<unknown>) {
    setBusyId(item.id);
    setError("");
    try {
      await action();
      reload();
    } catch (err) {
      setError(toMessage(err));
    } finally {
      setBusyId(null);
    }
  }

  const togglePublish = (a: StaffAnnouncement) =>
    act(a, () => api.post(`/api/v1/announcements/${a.id}/publish`, { published: !a.published }));
  const togglePin = (a: StaffAnnouncement) =>
    act(a, () => api.post(`/api/v1/announcements/${a.id}/pin`, { pinned: !a.pinned }));
  const remove = (a: StaffAnnouncement) => {
    if (!window.confirm(t("deleteConfirm"))) return;
    void act(a, async () => {
      await api.delete(`/api/v1/announcements/${a.id}`);
      if (editing !== null && editing !== "new" && editing.id === a.id) setEditing(null);
    });
  };

  const current = list?.key === key ? list : null;
  const pages = Math.max(1, Math.ceil((current?.total ?? 0) / PAGE_SIZE));

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-end gap-3">
        <label className="space-y-1 text-sm">
          <span className="block text-muted-foreground">{t("typeFilter")}</span>
          <NativeSelect className="h-9 w-44" value={type} onChange={(e) => { setType(e.target.value); setPage(1); }}>
            <option value="">{t("filterAll")}</option>
            {ANNOUNCEMENT_TYPES.map((v) => <option key={v} value={v}>{types(v)}</option>)}
          </NativeSelect>
        </label>
        <label className="space-y-1 text-sm">
          <span className="block text-muted-foreground">{t("statusFilter")}</span>
          <NativeSelect className="h-9 w-44" value={status} onChange={(e) => { setStatus(e.target.value as StatusFilter); setPage(1); }}>
            <option value="">{t("filterAll")}</option>
            <option value="published">{t("filterPublished")}</option>
            <option value="draft">{t("filterDrafts")}</option>
          </NativeSelect>
        </label>
        <Button className="ms-auto" onClick={() => setEditing("new")}>
          <Plus aria-hidden />
          {t("new")}
        </Button>
      </div>

      {editing !== null && (
        <AnnouncementEditor
          key={editing === "new" ? "new" : editing.id}
          announcement={editing === "new" ? null : editing}
          onSaved={(saved) => { setEditing(saved); reload(); }}
          onClose={() => setEditing(null)}
        />
      )}

      {error && <Alert variant="danger">{error}</Alert>}

      {!current ? (
        <div className="space-y-2" aria-hidden>{[0, 1, 2].map((i) => <Skeleton key={i} className="h-20 w-full" />)}</div>
      ) : current.items.length === 0 ? (
        <p className="rounded-xl border border-dashed p-8 text-center text-muted-foreground">{t("empty")}</p>
      ) : (
        <ul className="space-y-2">
          {current.items.map((a) => (
            <li key={a.id} className={cn("rounded-xl border bg-card p-3", a.pinned && "border-s-4 border-s-accent")}>
              <div className="flex flex-wrap items-center gap-2 text-xs">
                <AnnouncementTypeBadge type={a.type} />
                <span className={cn("rounded-full px-2 py-0.5 font-semibold", a.published ? FAMILY_TINT.success : FAMILY_TINT.neutral)}>
                  {a.published ? t("published") : t("draft")}
                </span>
                {a.pinned && <span className="inline-flex items-center gap-1 font-medium text-accent"><Pin className="size-3.5" aria-hidden />{t("pinned")}</span>}
                {a.attachment && <Paperclip className="size-3.5 text-muted-foreground" aria-label={t("attachment")} />}
                {a.activity_id && <span className="text-muted-foreground">{t("fromActivity")}</span>}
                <span className="rounded-full bg-muted px-2 py-0.5 text-muted-foreground">{t(`langs.${a.lang ?? "ar"}`)}</span>
                <span className="text-muted-foreground">
                  {a.published && a.published_at
                    ? t("publishedAt", { date: formatDate(a.published_at, locale) })
                    : t("createdAt", { date: formatDate(a.created_at, locale) })}
                </span>
              </div>
              <Bidi as="p" block text={a.title} lang={a.lang} className="mt-1.5 font-semibold" />
              <div className="mt-2 flex flex-wrap gap-1.5">
                <Button size="xs" variant="outline" onClick={() => setEditing(a)}>{t("edit")}</Button>
                <Button size="xs" variant={a.published ? "outline" : "default"} disabled={busyId === a.id} onClick={() => togglePublish(a)}>
                  {a.published ? t("unpublish") : t("publish")}
                </Button>
                <Button size="xs" variant="outline" disabled={busyId === a.id} onClick={() => togglePin(a)}>
                  {a.pinned ? t("unpin") : t("pin")}
                </Button>
                {a.published && (
                  <Button size="xs" variant="ghost" nativeButton={false} render={<Link href={`/news/${a.id}`} target="_blank" />}>
                    <ExternalLink aria-hidden />
                    {t("view")}
                  </Button>
                )}
                <Button size="xs" variant="destructive" disabled={busyId === a.id} onClick={() => remove(a)}>{t("delete")}</Button>
              </div>
            </li>
          ))}
        </ul>
      )}

      {pages > 1 && (
        <div className="flex items-center justify-center gap-3">
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>{tc("previous")}</Button>
          <span className="text-sm text-muted-foreground tabular-nums">{page} / {pages}</span>
          <Button variant="outline" size="sm" disabled={page >= pages} onClick={() => setPage((p) => p + 1)}>{tc("next")}</Button>
        </div>
      )}
    </div>
  );
}
