"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { api } from "@/lib/api";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Link } from "@/i18n/routing";
import { Bidi } from "@/components/i18n/bidi";
import { usePortalErrors } from "@/components/portal/use-portal-errors";
import { RIS_SYNC_API, type RISSyncReview } from "@/lib/portal/ris-sync";
import { ValueCell } from "./value-cell";

interface ReviewsListProps {
  reviews: RISSyncReview[];
  departments: Map<string, string>;
  onResolved: () => void;
}

/** Differences waiting for an admin: keep the portal value or take the RIS one. */
export function ReviewsList({ reviews, departments, onResolved }: ReviewsListProps) {
  const t = useTranslations("portal.risSync");
  const toMessage = usePortalErrors();
  const [busyId, setBusyId] = useState("");
  const [error, setError] = useState("");

  async function resolve(id: string, use: "portal" | "ris") {
    setBusyId(id);
    setError("");
    try {
      await api.post(RIS_SYNC_API.resolve(id), { use });
      onResolved();
    } catch (err) {
      setError(toMessage(err));
    } finally {
      setBusyId("");
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("reviewsTitle", { count: reviews.length })}</CardTitle>
        <p className="text-sm text-muted-foreground">{t("reviewsHint")}</p>
      </CardHeader>
      <CardContent className="space-y-3 overflow-x-auto">
        {error && <Alert variant="danger">{error}</Alert>}
        {reviews.length === 0 ? (
          <p className="py-6 text-center text-sm text-muted-foreground">{t("noReviews")}</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("researcher")}</TableHead>
                <TableHead>{t("field")}</TableHead>
                <TableHead>{t("inPortal")}</TableHead>
                <TableHead>{t("inRIS")}</TableHead>
                <TableHead>{t("decision")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {reviews.map((v) => (
                <TableRow key={v.id} className="align-top">
                  <TableCell className="whitespace-normal">
                    <Link href={`/researchers/${v.researcher_id}`} className="hover:underline"><Bidi text={v.researcher_name} /></Link>
                    {v.reason === "taken" && <div className="mt-1 text-xs text-status-warning-fg">{t("takenHint")}</div>}
                  </TableCell>
                  <TableCell>{t(`fields.${v.field}`)}</TableCell>
                  <TableCell className="max-w-64 whitespace-normal"><ValueCell value={v.portal_value} departments={departments} /></TableCell>
                  <TableCell className="max-w-64 whitespace-normal"><ValueCell value={v.ris_value} departments={departments} /></TableCell>
                  <TableCell>
                    <div className="flex flex-wrap gap-2">
                      <Button size="sm" disabled={busyId === v.id} onClick={() => resolve(v.id, "portal")}>{t("usePortal")}</Button>
                      <Button size="sm" variant="outline" disabled={busyId === v.id} onClick={() => resolve(v.id, "ris")}>{t("useRIS")}</Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}
