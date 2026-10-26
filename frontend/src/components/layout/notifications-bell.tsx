"use client";

import { useCallback, useEffect, useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import { Bell } from "lucide-react";
import { useRouter } from "@/i18n/routing";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useHydrated } from "@/hooks/use-hydrated";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

interface Notification {
  id: string;
  title_ar: string;
  title_en: string | null;
  body: string;
  type: string;
  link: string | null;
  is_read: boolean;
  created_at: string;
}

const POLL_MS = 60_000;

export function NotificationsBell() {
  const t = useTranslations("notifications");
  const locale = useLocale();
  const router = useRouter();
  const hydrated = useHydrated();
  const token = useAuth((state) => state.token);

  const [unread, setUnread] = useState(0);
  const [items, setItems] = useState<Notification[]>([]);
  const [open, setOpen] = useState(false);

  const refreshCount = useCallback(() => {
    api
      .get<{ unread: number }>("/api/v1/notifications/unread")
      .then((res) => setUnread(res.data?.unread ?? 0))
      .catch(() => undefined);
  }, []);

  useEffect(() => {
    if (!hydrated || !token) return;
    refreshCount();
    const timer = setInterval(refreshCount, POLL_MS);
    return () => clearInterval(timer);
  }, [hydrated, token, refreshCount]);

  useEffect(() => {
    if (!open) return;
    api
      .get<Notification[]>("/api/v1/notifications")
      .then((res) => setItems(res.data ?? []))
      .catch(() => undefined);
  }, [open]);

  if (!hydrated || !token) return null;

  async function openItem(n: Notification) {
    setOpen(false);
    if (!n.is_read) {
      api.put(`/api/v1/notifications/${n.id}/read`, {}).then(refreshCount).catch(() => undefined);
    }
    if (n.link) router.push(n.link);
  }

  async function markAll() {
    try {
      await api.put("/api/v1/notifications/read-all", {});
      setItems((prev) => prev.map((n) => ({ ...n, is_read: true })));
      setUnread(0);
    } catch {
      // keep the current state; the next poll corrects it
    }
  }

  return (
    <DropdownMenu open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger
        render={
          <Button variant="ghost" size="icon" aria-label={t("title")} className="relative" />
        }
      >
        <Bell className="size-4" />
        {unread > 0 && (
          <span className="absolute -end-0.5 -top-0.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-red-600 px-1 text-[10px] font-bold text-white">
            {unread > 99 ? "99+" : unread}
          </span>
        )}
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-96 p-0">
        <div className="flex items-center justify-between border-b px-3 py-2">
          <span className="text-sm font-semibold">{t("title")}</span>
          {unread > 0 && (
            <button type="button" onClick={markAll} className="text-xs text-primary hover:underline">
              {t("markAll")}
            </button>
          )}
        </div>
        <div className="max-h-96 overflow-auto">
          {items.length === 0 ? (
            <p className="p-6 text-center text-sm text-muted-foreground">{t("empty")}</p>
          ) : (
            items.map((n) => (
              <button
                key={n.id}
                type="button"
                onClick={() => openItem(n)}
                className={`block w-full border-b px-3 py-2 text-start last:border-b-0 hover:bg-muted ${
                  n.is_read ? "" : "bg-primary/5"
                }`}
              >
                <span className="flex items-start gap-2">
                  {!n.is_read && <span className="mt-1.5 size-2 shrink-0 rounded-full bg-primary" />}
                  <span className="min-w-0">
                    <span className="block truncate text-sm font-medium">
                      {locale === "ar" ? n.title_ar : (n.title_en ?? n.title_ar)}
                    </span>
                    {n.body && (
                      <span className="block truncate text-xs text-muted-foreground">{n.body}</span>
                    )}
                    <span className="block text-[11px] text-muted-foreground" dir="ltr">
                      {new Date(n.created_at).toLocaleString(
                        locale === "ar" ? "ar-IQ-u-nu-latn" : "en-GB"
                      )}
                    </span>
                  </span>
                </span>
              </button>
            ))
          )}
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
