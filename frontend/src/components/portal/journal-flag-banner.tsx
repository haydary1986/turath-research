"use client";

import { useTranslations } from "next-intl";
import { ShieldAlert, ShieldCheck, ShieldX } from "lucide-react";
import { FAMILY_TINT, SCREENING_FAMILY } from "@/lib/portal/status";
import type { JournalFlag } from "@/lib/portal/types";

const ICONS: Record<string, typeof ShieldX> = {
  predatory: ShieldX,
  suspended: ShieldAlert,
  accredited: ShieldCheck,
};

const BORDERS: Record<string, string> = {
  predatory: "border-status-danger/30",
  suspended: "border-status-warning/30",
  accredited: "border-status-success/30",
};

/** Watchlist verdict shown on verification requests and inside the request form. */
export function JournalFlagBanner({ flag }: { flag?: JournalFlag | null }) {
  const t = useTranslations("portal.journals");
  if (!flag?.matched || !flag.classification) return null;
  const family = SCREENING_FAMILY[flag.classification] ?? "warning";
  const Icon = ICONS[flag.classification] ?? ShieldAlert;

  return (
    <div
      className={`flex items-start gap-3 rounded-lg border p-3 text-sm ${BORDERS[flag.classification] ?? BORDERS.suspended} ${FAMILY_TINT[family]}`}
      role="alert"
    >
      <Icon className="mt-0.5 size-5 shrink-0" aria-hidden />
      <div className="min-w-0">
        <p className="font-semibold">
          {t(`flag.${flag.classification}`)}: <span dir="auto">{flag.name}</span>
        </p>
        {flag.note && <p className="mt-1 whitespace-pre-line">{flag.note}</p>}
        <p className="mt-1 text-xs opacity-80">{t("flag.hint")}</p>
      </div>
    </div>
  );
}
