"use client";

import { useTranslations } from "next-intl";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { requestTint } from "@/lib/portal/status";
import type { RequestStatus } from "@/lib/portal/types";

export function StatusBadge({ status }: { status: RequestStatus }) {
  const t = useTranslations("portal.statuses");
  return (
    <Badge variant="secondary" className={cn(requestTint(status))}>
      {t(status)}
    </Badge>
  );
}
