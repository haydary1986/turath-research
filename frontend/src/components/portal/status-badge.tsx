"use client";

import { useTranslations } from "next-intl";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import type { RequestStatus } from "@/lib/portal/types";

const STATUS_STYLE: Record<RequestStatus, string> = {
  draft: "bg-muted text-muted-foreground",
  pending_dean: "bg-amber-100 text-amber-800",
  pending: "bg-sky-100 text-sky-800",
  pending_vp: "bg-indigo-100 text-indigo-800",
  returned: "bg-orange-100 text-orange-800",
  approved: "bg-green-100 text-green-800",
  verified: "bg-green-100 text-green-800",
  completed: "bg-green-100 text-green-800",
  issued: "bg-blue-100 text-blue-800",
  awaiting_signatures: "bg-purple-100 text-purple-800",
  rejected: "bg-red-100 text-red-800",
  not_eligible: "bg-red-100 text-red-800",
};

export function StatusBadge({ status }: { status: RequestStatus }) {
  const t = useTranslations("portal.statuses");
  return (
    <Badge variant="secondary" className={cn(STATUS_STYLE[status])}>
      {t(status)}
    </Badge>
  );
}
