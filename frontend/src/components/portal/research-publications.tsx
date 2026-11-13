"use client";

import { useTranslations } from "next-intl";
import type { ProfilePublication } from "@/lib/portal/research-profile";
import { Alert } from "@/components/ui/alert";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table";

interface Props {
  publications: ProfilePublication[];
  total: number;
  hasOrcid: boolean;
  syncing: boolean;
}

export function ResearchPublications({ publications, total, hasOrcid, syncing }: Props) {
  const t = useTranslations("portal.profile");

  const emptyKey = !hasOrcid ? "emptyNoOrcid" : syncing ? "emptySyncing" : "emptyNoWorks";

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("pubsTitle")}</CardTitle>
        <p className="text-sm text-muted-foreground">{t("readOnly")}</p>
      </CardHeader>
      <CardContent>
        {publications.length === 0 ? (
          <Alert variant={!hasOrcid ? "warning" : "info"}>{t(emptyKey)}</Alert>
        ) : (
          <div className="space-y-3">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("colTitle")}</TableHead>
                  <TableHead>{t("colJournal")}</TableHead>
                  <TableHead>{t("colYear")}</TableHead>
                  <TableHead>{t("colDoi")}</TableHead>
                  <TableHead>{t("colCitations")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {publications.map((p) => (
                  <TableRow key={p.id}>
                    <TableCell className="min-w-64 whitespace-normal" dir="auto">{p.title}</TableCell>
                    <TableCell className="min-w-40 whitespace-normal" dir="auto">{p.journal ?? "—"}</TableCell>
                    <TableCell className="tabular-nums">{p.year ?? "—"}</TableCell>
                    <TableCell dir="ltr" className="text-start">
                      {p.doi ? (
                        <a
                          href={`https://doi.org/${encodeURI(p.doi)}`}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-primary underline-offset-4 hover:underline"
                        >
                          {p.doi}
                        </a>
                      ) : "—"}
                    </TableCell>
                    <TableCell className="tabular-nums">{p.citation_count}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            {total > publications.length && (
              <p className="text-xs text-muted-foreground">
                {t("showing", { shown: publications.length, total })}
              </p>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
