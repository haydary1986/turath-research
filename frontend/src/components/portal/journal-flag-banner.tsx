"use client";

import { useTranslations } from "next-intl";
import { ShieldAlert, ShieldCheck, ShieldX } from "lucide-react";
import type { JournalFlag } from "@/lib/portal/types";

const STYLE: Record<string, { box: string; Icon: typeof ShieldX }> = {
  predatory: { box: "border-red-300 bg-red-50 text-red-800", Icon: ShieldX },
  suspended: { box: "border-amber-300 bg-amber-50 text-amber-800", Icon: ShieldAlert },
  accredited: { box: "border-green-300 bg-green-50 text-green-800", Icon: ShieldCheck },
};

/** Watchlist verdict shown on verification requests and inside the request form. */
export function JournalFlagBanner({ flag }: { flag?: JournalFlag | null }) {
  const t = useTranslations("portal.journals");
  if (!flag?.matched || !flag.classification) return null;
  const { box, Icon } = STYLE[flag.classification] ?? STYLE.suspended;

  return (
    <div className={`flex items-start gap-3 rounded-lg border p-3 text-sm ${box}`} role="alert">
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
