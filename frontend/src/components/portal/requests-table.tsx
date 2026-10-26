"use client";

import { useTranslations } from "next-intl";
import { Link } from "@/i18n/routing";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { formatDate, requestTitle, type PortalRequest } from "@/lib/portal/types";
import { StatusBadge } from "./status-badge";

interface RequestsTableProps {
  items: PortalRequest[];
  locale: string;
  showApplicant?: boolean;
}

export function RequestsTable({ items, locale, showApplicant = false }: RequestsTableProps) {
  const t = useTranslations("portal");

  if (items.length === 0) {
    return <p className="py-6 text-center text-sm text-muted-foreground">{t("ui.noRequests")}</p>;
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t("fields.requestNo")}</TableHead>
          <TableHead>{t("fields.kind")}</TableHead>
          <TableHead>{t("fields.title")}</TableHead>
          {showApplicant && <TableHead>{t("fields.applicant")}</TableHead>}
          <TableHead>{t("fields.status")}</TableHead>
          <TableHead>{t("fields.submittedAt")}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {items.map((r) => (
          <TableRow key={r.id}>
            <TableCell dir="ltr" className="whitespace-nowrap">
              <Link href={`/portal/requests/${r.id}`} className="font-medium text-primary hover:underline">
                {r.request_no ?? "—"}
              </Link>
            </TableCell>
            <TableCell>{t(`kinds.${r.kind}`)}</TableCell>
            <TableCell className="max-w-xs truncate" dir="auto" title={requestTitle(r)}>
              {requestTitle(r)}
            </TableCell>
            {showApplicant && <TableCell>{r.applicant_name}</TableCell>}
            <TableCell>
              <StatusBadge status={r.status} />
            </TableCell>
            <TableCell dir="ltr" className="whitespace-nowrap">
              {formatDate(r.submitted_at ?? r.created_at, locale)}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
